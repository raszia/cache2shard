package cache2shard

import (
	"sync"
)

var SHARD_COUNT = 32

// A "thread" safe map of type string:Anything.
// To avoid lock bottlenecks this map is dived to several (SHARD_COUNT) map shards.
type ConcurrentMap[K comparable, V *CacheItem] struct {
	shards   []*ConcurrentMapShared[K, V]
	sharding func(key K) uint32
}

// A "thread" safe string to anything map.
type ConcurrentMapShared[K comparable, V *CacheItem] struct {
	items        map[K]V
	sync.RWMutex // Read Write mutex, guards access to internal map.
}

// Creates a new concurrent map.
func createCMAP[K comparable, V *CacheItem](sharding func(key K) uint32) ConcurrentMap[K, V] {
	m := ConcurrentMap[K, V]{
		sharding: sharding,
		shards:   make([]*ConcurrentMapShared[K, V], SHARD_COUNT),
	}
	for i := 0; i < SHARD_COUNT; i++ {
		m.shards[i] = &ConcurrentMapShared[K, V]{items: make(map[K]V)}
	}
	return m
}

// getShard returns shard under given key
func (m ConcurrentMap[K, V]) getShard(key K) *ConcurrentMapShared[K, V] {
	return m.shards[uint(m.sharding(key))%uint(SHARD_COUNT)]
}

// Sets the given value under the specified key.
func (m ConcurrentMap[K, V]) set(key K, value V) {
	// Get map shard.
	shard := m.getShard(key)
	shard.Lock()
	shard.items[key] = value
	shard.Unlock()
}

// get retrieves an element from map under given key.
func (m ConcurrentMap[K, V]) get(key K) (V, bool) {
	// Get shard
	shard := m.getShard(key)
	shard.RLock()
	// Get item from shard.
	val, ok := shard.items[key]
	shard.RUnlock()
	return val, ok
}

// count returns the number of elements within the map.
func (m ConcurrentMap[K, V]) count() int {
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
func (m ConcurrentMap[K, V]) has(key K) bool {
	// Get shard
	shard := m.getShard(key)
	shard.RLock()
	// See if element is within shard.
	_, ok := shard.items[key]
	shard.RUnlock()
	return ok
}

// remove removes an element from the map.
func (m ConcurrentMap[K, V]) remove(key K) {
	// Try to get shard.
	shard := m.getShard(key)
	shard.Lock()
	delete(shard.items, key)
	shard.Unlock()
}

// Used by the Iter & IterBuffered functions to wrap two variables together over a channel,
type Tuple[K comparable, V *CacheItem] struct {
	Key K
	Val V
}

// iterBuffered returns a buffered iterator which could be used in a for range loop.
func (m ConcurrentMap[K, V]) iterBuffered() <-chan Tuple[K, V] {
	chans := snapshot(m)
	total := 0
	for _, c := range chans {
		total += cap(c)
	}
	ch := make(chan Tuple[K, V], total)
	go fanIn(chans, ch)
	return ch
}

// Clear removes all items from map.
func (m ConcurrentMap[K, V]) clear() {
	for item := range m.iterBuffered() {
		m.remove(item.Key)
	}
}

// Returns a array of channels that contains elements in each shard,
// which likely takes a snapshot of `m`.
// It returns once the size of each buffered channel is determined,
// before all the channels are populated using goroutines.
func snapshot[K comparable, V *CacheItem](m ConcurrentMap[K, V]) (chans []chan Tuple[K, V]) {

	chans = make([]chan Tuple[K, V], SHARD_COUNT)
	wg := sync.WaitGroup{}
	wg.Add(SHARD_COUNT)
	// Foreach shard.
	for index, shard := range m.shards {
		go func(index int, shard *ConcurrentMapShared[K, V]) {
			// Foreach key, value pair.
			shard.RLock()
			chans[index] = make(chan Tuple[K, V], len(shard.items))
			wg.Done()
			for key, val := range shard.items {
				chans[index] <- Tuple[K, V]{key, val}
			}
			shard.RUnlock()
			close(chans[index])
		}(index, shard)
	}
	wg.Wait()
	return chans
}

// fanIn reads elements from channels `chans` into channel `out`
func fanIn[K comparable, V *CacheItem](chans []chan Tuple[K, V], out chan Tuple[K, V]) {
	wg := sync.WaitGroup{}
	wg.Add(len(chans))
	for _, ch := range chans {
		go func(ch chan Tuple[K, V]) {
			for t := range ch {
				out <- t
			}
			wg.Done()
		}(ch)
	}
	wg.Wait()
	close(out)
}

// Items returns all items as map[string]V
func (m ConcurrentMap[K, V]) Items() map[K]V {
	tmp := make(map[K]V)

	// Insert items to temporary map.
	for item := range m.iterBuffered() {
		tmp[item.Key] = item.Val
	}

	return tmp
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
