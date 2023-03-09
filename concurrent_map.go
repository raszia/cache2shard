package cache2shard

import (
	"sync"
)

var SHARD_COUNT = 32

// A "thread" safe map of type string:Anything.
// To avoid lock bottlenecks this map is dived to several (SHARD_COUNT) map shards.
type ConcurrentMap struct {
	shards   []*ConcurrentMapShared
	sharding func(key string) uint32
}

// A "thread" safe string to anything map.
type ConcurrentMapShared struct {
	items        map[string]*CacheItem
	sync.RWMutex // Read Write mutex, guards access to internal map.
}

// Creates a new concurrent map.
func createCMAP(sharding func(key string) uint32) ConcurrentMap {
	m := ConcurrentMap{
		sharding: sharding,
		shards:   make([]*ConcurrentMapShared, SHARD_COUNT),
	}
	for i := 0; i < SHARD_COUNT; i++ {
		m.shards[i] = &ConcurrentMapShared{items: make(map[string]*CacheItem)}
	}
	return m
}

// getShard returns shard under given key
func (m ConcurrentMap) getShard(key string) *ConcurrentMapShared {
	return m.shards[uint(m.sharding(key))%uint(SHARD_COUNT)]
}

// Sets the given value under the specified key.
func (m ConcurrentMap) set(key string, value *CacheItem) {
	// Get map shard.
	shard := m.getShard(key)
	shard.Lock()
	shard.items[key] = value
	shard.Unlock()
}

// get retrieves an element from map under given key.
func (m ConcurrentMap) get(key string) (*CacheItem, bool) {
	// Get shard
	shard := m.getShard(key)
	shard.RLock()
	// Get item from shard.
	val, ok := shard.items[key]
	shard.RUnlock()
	return val, ok
}

// count returns the number of elements within the map.
func (m ConcurrentMap) count() int {
	count := 0
	for i := 0; i < SHARD_COUNT; i++ {
		shard := m.shards[i]
		shard.RLock()
		count += len(shard.items)
		shard.RUnlock()
	}
	return count
}

// Looks up an item under specified key
func (m ConcurrentMap) has(key string) bool {
	// Get shard
	shard := m.getShard(key)
	shard.RLock()
	// See if element is within shard.
	_, ok := shard.items[key]
	shard.RUnlock()
	return ok
}

// remove removes an element from the map.
func (m ConcurrentMap) remove(key string) {
	// Try to get shard.
	shard := m.getShard(key)
	shard.Lock()
	delete(shard.items, key)
	shard.Unlock()
}

// Used by the Iter & IterBuffered functions to wrap two variables together over a channel,
type Tuple struct {
	key string
	Val *CacheItem
}

// iterBuffered returns a buffered iterator which could be used in a for range loop.
func (m ConcurrentMap) iterBuffered() <-chan Tuple {
	chans := snapshot(m)
	total := 0
	for _, c := range chans {
		total += cap(c)
	}
	ch := make(chan Tuple, total)
	go fanIn(chans, ch)
	return ch
}

// Returns a array of channels that contains elements in each shard,
// which likely takes a snapshot of `m`.
// It returns once the size of each buffered channel is determined,
// before all the channels are populated using goroutines.
func snapshot(m ConcurrentMap) (chans []chan Tuple) {

	chans = make([]chan Tuple, SHARD_COUNT)
	wg := sync.WaitGroup{}
	wg.Add(SHARD_COUNT)
	// Foreach shard.
	for index, shard := range m.shards {
		go func(index int, shard *ConcurrentMapShared) {
			// Foreach key, value pair.
			shard.RLock()
			chans[index] = make(chan Tuple, len(shard.items))
			wg.Done()
			for key, val := range shard.items {
				chans[index] <- Tuple{key, val}
			}
			shard.RUnlock()
			close(chans[index])
		}(index, shard)
	}
	wg.Wait()
	return chans
}

// fanIn reads elements from channels `chans` into channel `out`
func fanIn(chans []chan Tuple, out chan Tuple) {
	wg := sync.WaitGroup{}
	wg.Add(len(chans))
	for _, ch := range chans {
		go func(ch chan Tuple) {
			for t := range ch {
				out <- t
			}
			wg.Done()
		}(ch)
	}
	wg.Wait()
	close(out)
}

// fnv32 is an implementation of the FNV-1a hash.
// algorithm that takes a string key as input and returns
// a 32-bit unsigned integer hash value.
// The FNV-1a hash algorithm is a non-cryptographic hash function
// that produces a fast and evenly distributed hash value for a given input string.
// It works by multiplying the hash value by a prime number
// and then XORing the result with the next byte of the input string.
// This process is repeated for all bytes of the input string, resulting in a final hash value.
// In this specific implementation, the function initializes the hash value to
// a constant value (2166136261), multiplies it by another prime constant (16777619),
// and then XORs it with each byte of the input string.
// Finally, the function returns the resulting hash value as a 32-bit unsigned integer.
func fnv32(key string) uint32 {
	hash := uint32(2166136261)
	const prime32 = uint32(16777619)
	keyLength := len(key)
	for i := 0; i < keyLength; i++ {
		hash *= prime32
		hash ^= uint32(key[i])
	}
	return hash
}
