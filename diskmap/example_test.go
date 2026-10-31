package diskmap_test

import (
	"fmt"
	"os"

	"github.com/amp-labs/amp-common/diskmap"
)

func Example() {
	dir, err := os.MkdirTemp("", "diskmap-example")
	if err != nil {
		panic(err)
	}

	defer func() { _ = os.RemoveAll(dir) }()

	cache, err := diskmap.Open(dir, diskmap.WithMemoryLimit(16<<20))
	if err != nil {
		panic(err)
	}

	err = cache.Add("greeting", []byte("hello"))
	if err != nil {
		panic(err)
	}

	value, found, err := cache.Get("greeting")
	if err != nil {
		panic(err)
	}

	fmt.Println(string(value), found, cache.Size())

	// Close writes what is needed to reopen the map later.
	err = cache.Close()
	if err != nil {
		panic(err)
	}

	cache, err = diskmap.Open(dir)
	if err != nil {
		panic(err)
	}

	defer func() { _ = cache.Close() }()

	found, err = cache.Contains("greeting")
	fmt.Println(found, err)

	// Output:
	// hello true 1
	// true <nil>
}
