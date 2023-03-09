# cache2shard

[![Latest Release](https://img.shields.io/github/release/raszia/cache2shard.svg)](https://github.com/raszia/cache2shard/releases)
[![Build Status](https://github.com/raszia/cache2shard/workflows/build/badge.svg)](https://github.com/raszia/cache2shard/actions)
[![Coverage Status](https://coveralls.io/repos/github/raszia/cache2shard/badge.svg?branch=master)](https://coveralls.io/github/raszia/cache2shard?branch=master)
[![Go ReportCard](https://goreportcard.com/badge/raszia/cache2shard)](https://goreportcard.com/report/raszia/cache2shard)
[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](https://pkg.go.dev/github.com/raszia/cache2shard)

Concurrency-safe and fast golang caching library with expiration capabilities and sharding.

## Installation

Make sure you have a working Go environment (Go 1.18 or higher is required).
See the [install instructions](https://golang.org/doc/install.html).

To install cache2shard, simply run:

    go get github.com/raszia/cache2shard

To compile it from source:

    cd $GOPATH/src/github.com/raszia/cache2shard
    go get -u -v
    go build && go test -v

## Example

```go
package main

import (
 "github.com/raszia/cache2shard"
 "fmt"
 "time"
)

// Keys & values in cache2shard can be of arbitrary types, e.g. a struct.
type myStruct struct {
 text     string
 moreData []byte
}

func main() {
 // Accessing a new cache table for the first time will create it.
 cache := cache2shard.Cache("myCache")

 // We will put a new item in the cache. It will expire after
 // not being accessed via Value(key) for more than 5 seconds.
 val := myStruct{"This is a test!", []byte{}}
 cache.Add("someKey", 5*time.Second, &val)

 // Let's retrieve the item from the cache.
 res, err := cache.Value("someKey")
 if err == nil {
  fmt.Println("Found value in cache:", res.Data().(*myStruct).text)
 } else {
  fmt.Println("Error retrieving value from cache:", err)
 }

 // Wait for the item to expire in cache.
 time.Sleep(6 * time.Second)
 res, err = cache.Value("someKey")
 if err != nil {
  fmt.Println("Item is not cached (anymore).")
 }

 // Add another item that never expires.
 cache.Add("someKey", 0, &val)

 // cache2shard supports a few handy callbacks and loading mechanisms.
 cache.SetAboutToDeleteItemCallback(func(e *cache2shard.CacheItem) {
  fmt.Println("Deleting:", e.Key(), e.Data().(*myStruct).text, e.CreatedOn())
 })

 // Remove the item from the cache.
 cache.Delete("someKey")

 // And wipe the entire cache table.
 cache.Flush()
}
```

To run this example, go to examples/mycachedapp/ and run:

    go run mycachedapp.go

You can find a [few more examples here](https://github.com/raszia/cache2shard/tree/master/examples).
Also see our test-cases in cache_test.go for further working examples.
