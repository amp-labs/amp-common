package cmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	c := New(ctx, "echo", "hello")

	assert.NotNil(t, c)
	assert.NotNil(t, c.cmd)
	assert.Contains(t, c.cmd.Path, "echo") // Path may be absolute
	assert.Equal(t, []string{"echo", "hello"}, c.cmd.Args)
	assert.NotEmpty(t, c.cmd.Env) // Should inherit environment
}

func TestSetDir(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	c := New(ctx, "pwd").SetDir("/tmp")

	assert.Equal(t, "/tmp", c.cmd.Dir)
}

func TestSetStdin(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	input := strings.NewReader("test input")
	c := New(ctx, "cat").SetStdin(input)

	assert.Equal(t, input, c.cmd.Stdin)
}

func TestSetStdinBytes(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	input := []byte("test input")
	c := New(ctx, "cat").SetStdinBytes(input)

	assert.NotNil(t, c.cmd.Stdin)
}

func TestSetStdout(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	var buf bytes.Buffer
	c := New(ctx, "echo", "hello").SetStdout(&buf)

	assert.Equal(t, &buf, c.cmd.Stdout)
}

func TestSetStderr(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	var buf bytes.Buffer
	c := New(ctx, "ls", "/nonexistent").SetStderr(&buf)

	assert.Equal(t, &buf, c.cmd.Stderr)
}

func TestSetStdoutAndStderr(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	var buf bytes.Buffer
	c := New(ctx, "echo", "hello").SetStdoutAndStderr(&buf)

	assert.Equal(t, &buf, c.cmd.Stdout)
	assert.Equal(t, &buf, c.cmd.Stderr)
}

func TestSetStdoutObserver(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	var observed []byte

	c := New(ctx, "echo", "hello").SetStdoutObserver(func(output []byte) {
		observed = output
	})

	exitCode, err := c.Run()

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Contains(t, string(observed), "hello")
}

func TestSetStderrObserver(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	var observed []byte

	c := New(ctx, "sh", "-c", "echo error >&2").SetStderrObserver(func(output []byte) {
		observed = output
	})

	exitCode, err := c.Run()

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Contains(t, string(observed), "error")
}

func TestSetStdoutAndStderrObserver(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	var observed []byte

	c := New(ctx, "sh", "-c", "echo out; echo err >&2").
		SetStdoutAndStderrObserver(func(output []byte) {
			observed = output
		})

	exitCode, err := c.Run()

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Contains(t, string(observed), "out")
	assert.Contains(t, string(observed), "err")
}

func TestReplaceEnv(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	env := map[string]string{
		"KEY1": "value1",
		"KEY2": "value2",
	}
	c := New(ctx, "env").ReplaceEnv(env)

	assert.Len(t, c.cmd.Env, 2)
	assert.Contains(t, c.cmd.Env, "KEY1=value1")
	assert.Contains(t, c.cmd.Env, "KEY2=value2")
}

func TestAppendEnv(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	c := New(ctx, "env").AppendEnv("TEST_KEY", "test_value")

	assert.Contains(t, c.cmd.Env, "TEST_KEY=test_value")
}

func TestRun_Success(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	var buf bytes.Buffer
	c := New(ctx, "echo", "hello").SetStdout(&buf)

	exitCode, err := c.Run()

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Contains(t, buf.String(), "hello")
}

func TestRun_NonZeroExit(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	c := New(ctx, "sh", "-c", "exit 42")

	exitCode, _ := c.Run()
	assert.Equal(t, 42, exitCode)
}

func TestRun_CommandNotFound(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	c := New(ctx, "nonexistent-command-xyz")

	exitCode, err := c.Run()

	require.Error(t, err)
	assert.NotEqual(t, 0, exitCode)
}

func TestRun_ContextCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	c := New(ctx, "sleep", "10")

	// Cancel immediately
	cancel()

	exitCode, err := c.Run()

	require.Error(t, err)
	assert.NotEqual(t, 0, exitCode)
}

func TestRun_ContextTimeout(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	c := New(ctx, "sleep", "10")

	exitCode, err := c.Run()

	// Context timeout should result in an error or non-zero exit
	if err == nil {
		assert.NotEqual(t, 0, exitCode)
	}
}

func TestRun_WithStdin(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	var buf bytes.Buffer

	input := "test input"

	c := New(ctx, "cat").
		SetStdinBytes([]byte(input)).
		SetStdout(&buf)

	exitCode, err := c.Run()

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, input, buf.String())
}

func TestRun_WithWorkingDirectory(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	var buf bytes.Buffer

	c := New(ctx, "pwd").
		SetDir("/tmp").
		SetStdout(&buf)

	exitCode, err := c.Run()

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Contains(t, buf.String(), "/tmp")
}

func TestRun_WithCustomEnv(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	var buf bytes.Buffer

	c := New(ctx, "sh", "-c", "echo $CUSTOM_VAR").
		AppendEnv("CUSTOM_VAR", "custom_value").
		SetStdout(&buf)

	exitCode, err := c.Run()

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Contains(t, buf.String(), "custom_value")
}

func TestStatus_Success(t *testing.T) {
	t.Parallel()

	exitCode, err := status(nil)

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
}

func TestStatus_ExitError(t *testing.T) {
	t.Parallel()
	// Create a command that will fail
	ctx := t.Context()
	cmd := New(ctx, "sh", "-c", "exit 5")
	runErr := cmd.cmd.Run()

	exitCode, _ := status(runErr)

	assert.Equal(t, 5, exitCode)
}

func TestModSysProcAttr(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	c := New(ctx, "echo", "test")

	// Initially nil
	assert.Nil(t, c.cmd.SysProcAttr)

	// Set foreground
	c.SetForeground()
	assert.NotNil(t, c.cmd.SysProcAttr)
	assert.True(t, c.cmd.SysProcAttr.Foreground)

	// Set same process group
	c.SetSameProcessGroup()
	assert.True(t, c.cmd.SysProcAttr.Setpgid)
	assert.Equal(t, os.Getpid(), c.cmd.SysProcAttr.Pgid)
}

func TestChaining(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	var stdout, stderr bytes.Buffer

	c := New(ctx, "sh", "-c", "echo out; echo err >&2").
		SetDir("/tmp").
		AppendEnv("TEST", "value").
		SetStdout(&stdout).
		SetStderr(&stderr)

	exitCode, err := c.Run()

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Contains(t, stdout.String(), "out")
	assert.Contains(t, stderr.String(), "err")
}

func TestMultipleObservers(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	var observed1, observed2 []byte

	c := New(ctx, "echo", "hello")
	// Manually add multiple observers to test the finished slice
	var buf bytes.Buffer
	c.cmd.Stdout = &buf
	c.finished = append(c.finished, func() {
		observed1 = buf.Bytes()
	})
	c.finished = append(c.finished, func() {
		observed2 = buf.Bytes()
	})

	exitCode, err := c.Run()

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Contains(t, string(observed1), "hello")
	assert.Contains(t, string(observed2), "hello")
}

func TestRun_PipeOutput(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	// Test piping between commands using stdin/stdout
	var buf bytes.Buffer
	c := New(ctx, "sh", "-c", "echo 'line1\nline2\nline3' | grep line2").
		SetStdout(&buf)

	exitCode, err := c.Run()

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Contains(t, buf.String(), "line2")
	assert.NotContains(t, buf.String(), "line1")
}

func TestRun_DiscardOutput(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	c := New(ctx, "echo", "hello").
		SetStdout(io.Discard).
		SetStderr(io.Discard)

	exitCode, err := c.Run()

	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
}
