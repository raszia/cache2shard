package cache2shard_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/muesli/cache2go"
	"github.com/raszia/cache2shard"
)

func BenchmarkNotFoundAdd(b *testing.B) {
	table := cache2shard.CacheTable("testNotFoundAdd")
	testData := "testData"

	for j := 0; j < 100000; j++ {
		table.NotFoundAdd(fmt.Sprint(j), 0, testData)

	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < 100000; j++ {
			table.GetAndKeepAlive(fmt.Sprint(j))
		}
	}

}
func toKey(i int) string {
	return fmt.Sprintf("item:%d", i)
}
func BenchmarkCache2Shard(b *testing.B) {
	c := cache2shard.CacheTable("test")

	b.Run("Set", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			c.Add(toKey(i), 1*time.Minute, toKey(i))

		}
	})

	b.Run("Get", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			value, ok := c.GetAndKeepAlive(toKey(i))
			if ok {
				_ = value
			}
		}
	})

}
func BenchmarkCache2go(b *testing.B) {
	c := cache2go.Cache("test")

	b.Run("Set", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			c.Add(toKey(i), 1*time.Minute, toKey(i))

		}
	})

	b.Run("Get", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			value, err := c.Value(toKey(i))
			if err == nil {
				_ = value
			}
		}
	})

}
func BenchmarkAccessCount(b *testing.B) {
	// add 1000 items to the cache
	v := "testvalue"
	count := 1000
	table := cache2shard.CacheTable("testAccessCount")
	for i := 0; i < count; i++ {
		table.Add(fmt.Sprint(i), 10*time.Second, v)
	}
	// never access the first item, access the second item once, the third
	// twice and so on...
	for i := 0; i < count; i++ {
		for j := 0; j < i; j++ {
			table.GetAndKeepAlive(fmt.Sprint(i))
		}
	}

	b.ResetTimer()
	// check MostAccessed returns the items in correct order
	table.MostAccessed(int64(count))

}

func BenchmarkCache2shard(b *testing.B) {
	c := cache2shard.CacheTable("test")
	b.Run("SetOrginal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			c.Add(toKey(i), 1*time.Minute, toKey(i))

		}
	})

	b.Run("GetOrginal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			value, ok := c.GetAndKeepAlive(toKey(i))
			if ok {
				_ = value
			}
		}
	})
}
