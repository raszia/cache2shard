package main

import (
	"fmt"
	"time"

	"github.com/raszia/cache2shard"
)

func main() {
	cache := cache2shard.NewCacheTable("testTable")

	// This callback will be triggered every time a new item
	// gets added to the cache.
	cache.AddAddedItemCallback(func(entry *cache2shard.CacheItem) {
		fmt.Println("Added Callback 1:", entry.Key(), entry.Data(), entry.CreatedOn())
	})
	cache.AddAddedItemCallback(func(entry *cache2shard.CacheItem) {
		fmt.Println("Added Callback 2:", entry.Key(), entry.Data(), entry.CreatedOn())
	})
	// This callback will be triggered every time an item
	// is about to be removed from the cache.
	cache.AddAboutToDeleteItemCallback(func(entry *cache2shard.CacheItem) {
		fmt.Println("Deleting:", entry.Key(), entry.Data(), entry.CreatedOn())
	})

	// Caching a new item will execute the AddedItem callback.
	cache.Add("someKey", 0, "This is a test!")

	// Let's retrieve the item from the cache
	res, ok := cache.GetAndKeepAlive("someKey")
	if !ok {
		fmt.Println("Error retrieving value from cache")
	}

	fmt.Println("Found value in cache:", res.Data())

	// Deleting the item will execute the AboutToDeleteItem callback.
	cache.Delete("someKey")

	cache.RemoveAddedItemCallbacks()
	// Caching a new item that expires in 3 seconds
	res = cache.Add("anotherKey", 3*time.Second, "This is another test")

	// This callback will be triggered when the item is about to expire
	res.AddAboutToExpireCallback(func(key string) {
		fmt.Println("About to expire:", key)
	})

	time.Sleep(5 * time.Second)
}
