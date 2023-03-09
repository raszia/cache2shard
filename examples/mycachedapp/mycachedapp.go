package main

import (
	"fmt"
	"time"

	"github.com/raszia/cache2shard"
)

// Keys & values in cache2shard can be of arbitrary types, e.g. a struct.
type myStruct struct {
	text     string
	moreData []byte
}

func main() {
	// Accessing a new cache table for the first time will create it.
	cache := cache2shard.CacheTable("myCache")

	// We will put a new item in the cache. It will expire after
	// not being accessed via Value(key) for more than 5 seconds.
	val := myStruct{"This is a test!", []byte{}}
	cache.Add("someKey", 5*time.Second, &val)

	// Let's retrieve the item from the cache.
	res, ok := cache.GetAndKeepAlive("someKey")
	if ok {
		fmt.Println("Found value in cache:", res.Data().(*myStruct).text)
	} else {
		fmt.Println("Error retrieving value from cache")
	}

	// Wait for the item to expire in cache.
	time.Sleep(6 * time.Second)
	_, ok = cache.GetAndKeepAlive("someKey")
	if !ok {
		fmt.Println("Item is not cached (anymore).")
	}

	// Add another item that never expires.
	cache.Add("someKey", 0, &val)

	// cache2shard supports a few handy callbacks and loading mechanisms.
	cache.AddAboutToDeleteItemCallback(func(e *cache2shard.CacheItem) {
		fmt.Println("Deleting:", e.Key(), e.Data().(*myStruct).text, e.CreatedOn())
	})

	// Remove the item from the cache.
	cache.Delete("someKey")

	// And wipe the entire cache table.
	cache.Flush()
}
