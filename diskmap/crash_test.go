package diskmap

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// crash abandons the map the way a crashing process would: no checkpoint,
// and anything not yet handed to the operating system is lost.
func (m *Map) crash() {
	m.closeMu.Lock()
	defer m.closeMu.Unlock()

	if m.closed.Swap(true) {
		return
	}

	// Failing the log wakes every waiter and stops all further writes, the
	// way the process vanishing would.
	m.log.mu.Lock()
	_ = m.log.fail(errCrashed)
	m.log.queue = nil
	m.log.mu.Unlock()

	m.shutdown()
}

var errCrashed = errors.New("simulated crash")

// crashOptions make segments rotate, compaction run and checkpoints happen
// often, so a crash lands among all of them.
func crashOptions(extra ...Option) []Option {
	return append([]Option{
		WithMemoryLimit(2 << 20),
		WithSegmentSize(minSegmentSize),
		WithCheckpointInterval(minCheckpointInterval),
		WithCompactionThreshold(0.3),
		WithShards(4),
	}, extra...)
}

// versionedValue is the value written by operation version to key slot.
func versionedValue(slot, version int) []byte {
	return fmt.Appendf(nil, "k%d=%d|%s", slot, version, strings.Repeat("v", version%700))
}

func parseVersion(value []byte) (slot, version int, ok bool) {
	head, _, _ := strings.Cut(string(value), "|")

	slotPart, versionPart, found := strings.Cut(strings.TrimPrefix(head, "k"), "=")
	if !found {
		return 0, 0, false
	}

	slot, err1 := strconv.Atoi(slotPart)
	version, err2 := strconv.Atoi(versionPart)

	return slot, version, err1 == nil && err2 == nil
}

// checkPrefix checks that dmap holds exactly the effect of operations
// 0..last of the sequence that wrote slot i%slots = version i, removing the
// key instead when i%7 == 6, and that last is at least acked. It returns
// last.
func checkPrefix(t *testing.T, dmap *Map, slots, acked int) int {
	t.Helper()

	last := -1

	require.NoError(t, dmap.Range(func(key string, value []byte) bool {
		slot, version, ok := parseVersion(value)
		require.True(t, ok, "key %s has value %q", key, value)
		require.Equal(t, "k"+strconv.Itoa(slot), key)

		last = max(last, version)

		return true
	}))

	// A removal may be the last operation applied.
	for i := last + 1; i < last+slots && i%7 == 6; i++ {
		found, err := dmap.Contains("k" + strconv.Itoa(i%slots))
		require.NoError(t, err)

		if !found {
			last = i
		}
	}

	require.GreaterOrEqual(t, last, acked, "an acknowledged write was lost")

	want := make(map[string][]byte)

	for i := 0; i <= last; i++ {
		key := "k" + strconv.Itoa(i%slots)
		if i%7 == 6 {
			delete(want, key)
		} else {
			want[key] = versionedValue(i%slots, i)
		}
	}

	checkModel(t, dmap, want)

	return last
}

// writeSequence runs operations from..to-1 of the sequence checkPrefix
// expects.
func writeSequence(t *testing.T, dmap *Map, slots, from, to int) {
	t.Helper()

	for i := from; i < to; i++ {
		key := "k" + strconv.Itoa(i%slots)
		if i%7 == 6 {
			require.NoError(t, dmap.Remove(key))
		} else {
			require.NoError(t, dmap.Add(key, versionedValue(i%slots, i)))
		}
	}
}

func TestCrashKeepsAcknowledgedWrites(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name     string
		opts     []Option
		buffered bool
		ops      int
	}{
		{name: "default", ops: 4000},
		// Every write is an fsync, which on macOS flushes the drive's cache.
		{name: "sync writes", opts: []Option{WithSyncWrites()}, ops: 400},
		{name: "buffered", opts: []Option{WithBufferedWrites()}, buffered: true, ops: 4000},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			opts := crashOptions(test.opts...)

			const slots = 500

			next := 0

			for round := range 4 {
				dmap, err := Open(dir, opts...)
				require.NoError(t, err)

				if round > 0 {
					// Buffered writes may lose a tail, but never a write
					// from before it.
					acked := next - 1
					if test.buffered {
						acked = -1
					}

					next = checkPrefix(t, dmap, slots, acked) + 1
				}

				writeSequence(t, dmap, slots, next, next+test.ops)
				next += test.ops

				assert.Positive(t, dmap.Stats().Checkpoints)
				dmap.crash()
			}

			dmap, err := Open(dir, opts...)
			require.NoError(t, err)

			if test.buffered {
				checkPrefix(t, dmap, slots, -1)
			} else {
				checkPrefix(t, dmap, slots, next-1)
			}

			require.NoError(t, dmap.Close())
		})
	}
}

// lastVersion returns the newest operation whose effect the map shows.
func lastVersion(t *testing.T, dmap *Map) int {
	t.Helper()

	last := -1

	require.NoError(t, dmap.Range(func(_ string, value []byte) bool {
		_, version, ok := parseVersion(value)
		require.True(t, ok)

		last = max(last, version)

		return true
	}))

	return last
}

func TestRecoveryStopsAtTornRecord(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	const slots = 300

	dmap, err := Open(dir, crashOptions(WithCheckpointInterval(64<<20))...)
	require.NoError(t, err)
	require.NoError(t, dmap.Close())

	dmap, err = Open(dir, crashOptions(WithCheckpointInterval(64<<20))...)
	require.NoError(t, err)

	writeSequence(t, dmap, slots, 0, 900)
	dmap.crash()

	// Tear the newest record of the newest foreground segment.
	newest := newestFile(t, dir, segmentSuffix)
	info, err := os.Stat(newest)
	require.NoError(t, err)
	require.NoError(t, os.Truncate(newest, info.Size()-3))

	dmap, err = Open(dir, crashOptions()...)
	require.NoError(t, err)

	last := lastVersion(t, dmap)
	assert.Less(t, last, 899, "the torn record must not be applied")
	checkPrefix(t, dmap, slots, last)
	require.NoError(t, dmap.Close())
}

func newestFile(t *testing.T, dir, suffix string) string {
	t.Helper()

	matches, err := filepath.Glob(filepath.Join(dir, "*"+suffix))
	require.NoError(t, err)
	require.NotEmpty(t, matches)

	newest := matches[0]

	for _, path := range matches {
		info, err := os.Stat(path)
		require.NoError(t, err)

		if info.Size() > 0 && path > newest {
			newest = path
		}
	}

	return newest
}

func TestRecoveryIgnoresPagesOutsideTheCheckpoint(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	const slots = 3000

	dmap, err := Open(dir, crashOptions(WithCheckpointInterval(4<<20))...)
	require.NoError(t, err)

	writeSequence(t, dmap, slots, 0, 30_000)
	dmap.crash()

	// Scribble over every index page the checkpoint does not own. A
	// checkpoint must never depend on them.
	saved, err := readMeta(dir)
	require.NoError(t, err)

	garbage := bytes.Repeat([]byte{0xa5}, pageSize)

	for shardID, state := range saved.shards {
		owned := make(map[uint32]bool)
		for _, slot := range state.pageTable {
			owned[slot] = true
		}

		path := indexPath(dir, shardID)
		info, err := os.Stat(path)
		require.NoError(t, err)

		file, err := os.OpenFile(path, os.O_WRONLY, 0) //nolint:gosec // Test file.
		require.NoError(t, err)

		for slot := range uint32(info.Size() / pageSize) { //nolint:gosec // Small.
			if !owned[slot] {
				_, err = file.WriteAt(garbage, int64(slot)*pageSize)
				require.NoError(t, err)
			}
		}

		require.NoError(t, file.Close())
	}

	dmap, err = Open(dir, crashOptions()...)
	require.NoError(t, err)
	checkPrefix(t, dmap, slots, 29_999)
	require.NoError(t, dmap.Close())
}

func TestClearSurvivesCrash(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	dmap, err := Open(dir, crashOptions()...)
	require.NoError(t, err)

	writeSequence(t, dmap, 100, 0, 2000)
	require.NoError(t, dmap.Clear())
	require.NoError(t, dmap.Add("after", []byte("clear")))
	dmap.crash()

	dmap, err = Open(dir, crashOptions()...)
	require.NoError(t, err)
	checkModel(t, dmap, map[string][]byte{"after": []byte("clear")})
	require.NoError(t, dmap.Close())
}

// TestCrashChild is the child process of TestKilledProcessKeepsWrites. It
// writes until it is killed, printing the number of each acknowledged
// operation.
func TestCrashChild(t *testing.T) {
	t.Parallel()

	dir := os.Getenv("DISKMAP_CRASH_DIR")
	if dir == "" {
		t.Skip("run by TestKilledProcessKeepsWrites")
	}

	dmap, err := Open(dir, crashOptions()...)
	if err != nil {
		fmt.Println("open:", err) //nolint:forbidigo // Talking to the parent.
		os.Exit(1)
	}

	rng := rand.New(rand.NewPCG(1, 1)) //nolint:gosec // Test data.
	start, _ := strconv.Atoi(os.Getenv("DISKMAP_CRASH_START"))

	for i := start; ; i++ {
		key := "k" + strconv.Itoa(i%crashSlots)

		if i%7 == 6 {
			err = dmap.Remove(key)
		} else {
			err = dmap.Add(key, versionedValue(i%crashSlots, i))
		}

		if err != nil {
			fmt.Println("write:", err) //nolint:forbidigo // Talking to the parent.
			os.Exit(1)
		}

		fmt.Println(i) //nolint:forbidigo // Talking to the parent.

		if rng.IntN(5000) == 0 {
			_ = dmap.Sync()
		}
	}
}

const crashSlots = 2000

func TestKilledProcessKeepsWrites(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("starts a child process")
	}

	dir := t.TempDir()
	next := 0

	for round := range 3 {
		acked := runAndKill(t, dir, next, next+15_000)

		dmap, err := Open(dir, crashOptions()...)
		require.NoError(t, err, "round %d", round)

		next = checkPrefix(t, dmap, crashSlots, acked) + 1

		require.NoError(t, dmap.Close())
	}
}

// runAndKill starts the child at operation start, kills it once it has
// acknowledged operation target, and returns the last operation it
// acknowledged.
func runAndKill(t *testing.T, dir string, start, target int) int {
	t.Helper()

	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestCrashChild$") //nolint:gosec,lll // Re-runs this test binary.

	cmd.Env = append(os.Environ(), "DISKMAP_CRASH_DIR="+dir, "DISKMAP_CRASH_START="+strconv.Itoa(start))

	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())

	acked := -1
	scanner := bufio.NewScanner(stdout)

	for scanner.Scan() {
		n, err := strconv.Atoi(scanner.Text())
		if err != nil {
			continue
		}

		acked = n
		if acked >= target {
			break
		}
	}

	require.NoError(t, cmd.Process.Kill())

	_ = cmd.Wait()

	require.GreaterOrEqual(t, acked, target, "the child stopped early")

	return acked
}
