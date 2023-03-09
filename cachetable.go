package cache2shard

import (
	"sort"
	"sync"
	"time"
)

// CacheTable is a table within the cache
type CacheTable struct {
	muRW sync.RWMutex
	// The table's name.
	name string
	// All cached items.
	items ConcurrentMap[string, *CacheItem]

	// Timer responsible for triggering cleanup.
	cleanupTimer *time.Timer
	// Current timer duration.
	cleanupInterval time.Duration

	// Callback method triggered when trying to load a non-existing key.
	loadData func(key string, args ...any) (*CacheItem, bool)
	// Callback method triggered when adding a new item to the cache.
	addedItemCallback []func(item *CacheItem)
	// Callback method triggered before deleting an item from the cache.
	aboutToDeleteItemcallback []func(item *CacheItem)
}

// Count returns how many items are currently stored in the cache.
func (table *CacheTable) Count() int {
	return table.items.count()
}

// Foreach all items
func (table *CacheTable) Foreach(trans func(key string, item *CacheItem)) {

	for t := range table.items.iterBuffered() {
		trans(t.Key, t.Val)
	}

}

// SetDataLoader configures a data-loader callback, which will be called when
// trying to access a non-existing key. The key and 0...n additional arguments
// are passed to the callback function.
func (table *CacheTable) SetDataLoader(f func(string, ...any) (*CacheItem, bool)) {
	table.muRW.Lock()
	defer table.muRW.Unlock()
	table.loadData = f
}

// AddAddedItemCallback appends a new callback to the addedItem queue
func (table *CacheTable) AddAddedItemCallback(f func(*CacheItem)) {
	table.muRW.Lock()
	defer table.muRW.Unlock()
	table.addedItemCallback = append(table.addedItemCallback, f)
}

// RemoveAddedItemCallbacks empties the added item callback queue
func (table *CacheTable) RemoveAddedItemCallbacks() {
	table.muRW.Lock()
	defer table.muRW.Unlock()
	table.addedItemCallback = nil
}

// AddAboutToDeleteItemCallback appends a new callback to the AboutToDeleteItem queue
func (table *CacheTable) AddAboutToDeleteItemCallback(f func(*CacheItem)) {
	table.muRW.Lock()
	defer table.muRW.Unlock()
	table.aboutToDeleteItemcallback = append(table.aboutToDeleteItemcallback, f)
}

// RemoveAboutToDeleteItemCallback empties the about to delete item callback queue
func (table *CacheTable) RemoveAboutToDeleteItemCallback() {
	table.muRW.Lock()
	defer table.muRW.Unlock()

	table.aboutToDeleteItemcallback = nil
}

// Expiration check loop, triggered by a self-adjusting timer.
func (table *CacheTable) expirationCheck() {

	if table.cleanupTimer != nil {
		table.cleanupTimer.Stop()
	}

	// To be more accurate with timers, we would need to update 'now' on every
	// loop iteration. Not sure it's really efficient though.
	now := time.Now()
	smallestDuration := 0 * time.Second
	for t := range table.items.iterBuffered() {
		// Cache values so we don't keep blocking the mutex.
		// t.Val.RWMutex.RLock()

		// t.Val.RWMutex.RUnlock()

		if t.Val.lifeSpan == 0 {
			continue
		}
		if now.Sub(t.Val.accessedOn) >= t.Val.lifeSpan {
			// Item has excessed its lifespan.
			table.deleteInternal(t.Key, t.Val)
		} else {
			// Find the item chronologically closest to its end-of-lifespan.
			if smallestDuration == 0 || t.Val.lifeSpan-now.Sub(t.Val.accessedOn) < smallestDuration {
				smallestDuration = t.Val.lifeSpan - now.Sub(t.Val.accessedOn)
			}
		}
	}

	// Setup the interval for the next cleanup run.
	table.cleanupInterval = smallestDuration
	if smallestDuration > 0 {
		table.cleanupTimer = time.AfterFunc(smallestDuration, func() {
			go table.expirationCheck()
		})
	}
}

func (table *CacheTable) addInternal(item *CacheItem) {

	table.items.set(item.key, item)

	// Trigger callback after adding an item to cache.
	table.muRW.RLock()
	for _, callback := range table.addedItemCallback {
		callback(item)
	}
	table.muRW.RUnlock()

	// If we haven't set up any expiration check timer or found a more imminent item.
	if item.lifeSpan > 0 && (table.cleanupInterval == 0 || item.lifeSpan < table.cleanupInterval) {
		table.expirationCheck()
	}

}

// Add adds a key/value pair to the cache.
// Parameter key is the item's cache-key.
// Parameter lifeSpan determines after which time period without an access the item
// will get removed from the cache.
// Parameter data is the item's value.
func (table *CacheTable) Add(key string, lifeSpan time.Duration, data any) *CacheItem {
	item := NewCacheItem(key, lifeSpan, data)

	table.addInternal(item)
	return item
}

func (table *CacheTable) deleteInternal(key string, item *CacheItem) {

	// Trigger callbacks before deleting an item from cache.
	table.muRW.RLock()
	for _, callback := range table.aboutToDeleteItemcallback {
		callback(item)
	}
	table.muRW.RUnlock()

	item.RLock()
	for _, callback := range item.aboutToExpireCallback {
		callback(key)
	}
	item.RUnlock()

	table.items.remove(key)

}

// Delete an item from the cache.
func (table *CacheTable) Delete(key string) bool {
	if item, ok := table.items.get(key); ok {
		table.deleteInternal(key, item)
		return true
	}
	return false
}

// Exists returns whether an item exists in the cache. Unlike the Value method
// Exists neither tries to fetch data via the loadData callback nor does it
// keep the item alive in the cache.
func (table *CacheTable) Exists(key string) bool {
	return table.items.has(key)
}

// NotFoundAdd checks whether an item is not yet cached. Unlike the Exists
// method this also adds data if the key could not be found.
func (table *CacheTable) NotFoundAdd(key string, lifeSpan time.Duration, data any) bool {

	item := NewCacheItem(key, lifeSpan, data)

	if ok := table.Exists(key); !ok {
		table.addInternal(item)
		return true
	}

	return false
}

// GetAndKeepAlive returns an item from the cache and marks it to be kept alive. You can
// pass additional arguments to your DataLoader callback function.
func (table *CacheTable) GetAndKeepAlive(key string, args ...any) (*CacheItem, bool) {
	if item, ok := table.items.get(key); ok {
		item.KeepAlive() // Update access counter and timestamp.
		return item, true
	}

	// Item doesn't exist in cache. Try and fetch it with a data-loader.
	if table.loadData != nil {
		if item, ok := table.loadData(key, args...); ok {
			table.Add(key, item.LifeSpan(), item.Data())
			return item, true
		}

	}

	return nil, false
}

// Flush deletes all items from this cache table.
func (table *CacheTable) Flush() {

	table.items = createCMAP(fnv32)

	table.cleanupInterval = 0
	if table.cleanupTimer != nil {
		table.cleanupTimer.Stop()
	}
}

// CacheItemPair maps key to access counter
type CacheItemPair struct {
	Key         string
	AccessCount int64
}

// CacheItemPairList is a slice of CacheItemPairs that implements sort.
// Interface to sort by AccessCount.
type CacheItemPairList []CacheItemPair

func (p CacheItemPairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p CacheItemPairList) Len() int           { return len(p) }
func (p CacheItemPairList) Less(i, j int) bool { return p[i].AccessCount > p[j].AccessCount }

// MostAccessed returns the most accessed items in this cache table
func (table *CacheTable) MostAccessed(count int64) []*CacheItem {

	pairList := CacheItemPairList{}

	for items := range table.items.iterBuffered() {
		pairList = append(pairList, CacheItemPair{items.Key, items.Val.accessCount})
	}

	sort.Sort(pairList)

	var itemsList []*CacheItem
	c := int64(0)
	for _, pair := range pairList {
		if c >= count {
			break
		}

		if item, ok := table.items.get(pair.Key); ok {
			itemsList = append(itemsList, item)
		}
		c++
	}

	return itemsList
}
