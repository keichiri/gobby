package storage

import (
	"sort"
	"sync"
	"time"
)

type cache struct {
	maxCount int
	records  map[int]*cacheRecord
	mx       sync.Mutex
}

func newCache(maxCount int) *cache {
	return &cache{
		maxCount: maxCount,
		records:  make(map[int]*cacheRecord),
	}
}

func (c *cache) Put(index int, data []byte) {
	c.mx.Lock()
	if len(c.records) == c.maxCount {
		c.purge()
	}
	newRecord := &cacheRecord{
		index: index,
		data:  data,
		ts:    time.Now(),
	}
	c.records[index] = newRecord
	c.mx.Unlock()
}

func (c *cache) Get(index int) []byte {
	c.mx.Lock()
	record, exists := c.records[index]
	if !exists {
		c.mx.Unlock()
		return nil
	}
	record.ts = time.Now()
	c.mx.Unlock()
	return record.data
}

func (c *cache) purge() {
	allRecords := make([]*cacheRecord, 0, len(c.records))
	for _, record := range c.records {
		allRecords = append(allRecords, record)
	}

	sort.SliceStable(allRecords, func(i int, j int) bool {
		record1 := allRecords[i]
		record2 := allRecords[j]
		return record1.ts.Before(record2.ts)
	})
	toDelete := len(c.records)/4 + 1
	for _, record := range allRecords[:toDelete] {
		delete(c.records, record.index)
	}
}

type cacheRecord struct {
	index int
	data  []byte
	ts    time.Time
}
