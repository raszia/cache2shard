package main

import (
	"fmt"
	"strconv"

	"github.com/raszia/cache2shard"
)

func main() {
	cache := cache2shard.CacheTable("myCache")

	// The data loader gets called automatically whenever something
	// tries to retrieve a non-existing key from the cache.
	f := func(key string, args ...any) (*cache2shard.CacheItem, bool) {
		// Apply some clever loading logic here, e.g. read values for
		// this key from database, network or file.
		val := "This is a test with key " + key

		// This helper method creates the cached item for us. Yay!
		item := cache2shard.NewCacheItem(key, 0, val)
		return item, true
	}
	cache.SetDataLoader(f)

	// Let's retrieve a few auto-generated items from the cache.
	for i := 0; i < 10; i++ {
		res, ok := cache.GetAndKeepAlive("someKey_" + strconv.Itoa(i))
		if ok {
			fmt.Println("Found value in cache:", res.Data())
		} else {
			fmt.Println("Error retrieving value from cache")
		}
	}
}
