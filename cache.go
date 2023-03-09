package cache2shard

import (
	"sync"
)

type cacheMap struct {
	muRW     sync.RWMutex
	cacheMap map[string]*cacheTable
}

var cache = &cacheMap{
	muRW:     sync.RWMutex{},
	cacheMap: make(map[string]*cacheTable),
}

// CacheTable returns the existing cache table with given name or creates a new one
// if the table does not exist yet.
func CacheTable(tableName string) *cacheTable {
	table, ok := cache.get(tableName)
	if ok {
		return table
	}

	table = &cacheTable{
		name:  tableName,
		items: createCMAP(fnv32),
	}
	cache.set(tableName, table)

	return table
}

func DeleteCacheTable(tableName string) bool {
	return cache.delete(tableName)
}

func (c *cacheMap) set(tableName string, table *cacheTable) *cacheMap {
	c.muRW.Lock()
	defer c.muRW.Unlock()

	c.cacheMap[tableName] = table
	return c
}

func (c *cacheMap) get(tableName string) (*cacheTable, bool) {
	c.muRW.RLock()
	defer c.muRW.RUnlock()

	value, ok := c.cacheMap[tableName]
	return value, ok
}

func (c *cacheMap) delete(tableName string) bool {
	c.muRW.Lock()
	defer c.muRW.Unlock()
	if _, ok := c.cacheMap[tableName]; ok {
		delete(c.cacheMap, tableName)
		return true
	}

	return false

}
