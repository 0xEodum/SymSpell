package internal

import (
	"container/list"
	"symspell/pkg/items"
)

type topCache struct {
	capacity int
	ll       *list.List
	cache    map[string]*list.Element
}

type cacheEntry struct {
	key string
	val items.SuggestItem
}

func newTopCache(capacity int) *topCache {
	return &topCache{
		capacity: capacity,
		ll:       list.New(),
		cache:    make(map[string]*list.Element),
	}
}

func (c *topCache) Get(key string) (items.SuggestItem, bool) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		return ele.Value.(cacheEntry).val, true
	}
	return items.SuggestItem{}, false
}

func (c *topCache) Add(key string, val items.SuggestItem) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele)
		ele.Value = cacheEntry{key: key, val: val}
		return
	}
	ele := c.ll.PushFront(cacheEntry{key: key, val: val})
	c.cache[key] = ele
	if c.ll.Len() > c.capacity {
		if last := c.ll.Back(); last != nil {
			c.ll.Remove(last)
			entry := last.Value.(cacheEntry)
			delete(c.cache, entry.key)
		}
	}
}
