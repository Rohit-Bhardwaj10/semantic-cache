package cache

import (
	"container/list"
	"sync"
	"time"
)

type l1Item struct {
	key        string
	value      string
	tenantID   string
	size       int64
	expiration time.Time
}

// L1Cache is an in-memory LRU cache with a byte-based memory budget.
type L1Cache struct {
	mu         sync.RWMutex
	maxBytes   int64
	currBytes  int64
	items      map[string]*list.Element
	evictList  *list.List
}

func NewL1Cache(maxBytes int64) *L1Cache {
	return &L1Cache{
		maxBytes:  maxBytes,
		currBytes: 0,
		items:     make(map[string]*list.Element),
		evictList: list.New(),
	}
}

// Get retrieves a value from the L1 cache if it exists and hasn't expired.
func (c *L1Cache) Get(tenantID, query string) (string, bool) {
	key := tenantID + ":" + query
	c.mu.Lock()
	defer c.mu.Unlock()

	if ent, ok := c.items[key]; ok {
		item := ent.Value.(*l1Item)

		// Check expiration
		if !item.expiration.IsZero() && time.Now().After(item.expiration) {
			c.removeElement(ent)
			return "", false
		}

		c.evictList.MoveToFront(ent)
		return item.value, true
	}

	return "", false
}

// Set adds or updates a value in the L1 cache.
func (c *L1Cache) Set(tenantID, query, value string, ttl time.Duration) {
	key := tenantID + ":" + query
	size := int64(len(key) + len(value))
	var expiration time.Time
	if ttl > 0 {
		expiration = time.Now().Add(ttl)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// If entry already exists, update it
	if ent, ok := c.items[key]; ok {
		c.evictList.MoveToFront(ent)
		item := ent.Value.(*l1Item)
		c.currBytes -= item.size
		item.value = value
		item.size = size
		item.expiration = expiration
		c.currBytes += size
	} else {
		// Create new entry
		ent := &l1Item{
			key:        key,
			value:      value,
			tenantID:   tenantID,
			size:       size,
			expiration: expiration,
		}
		element := c.evictList.PushFront(ent)
		c.items[key] = element
		c.currBytes += size
	}

	// Evict if over budget
	for c.currBytes > c.maxBytes && c.evictList.Len() > 0 {
		c.removeOldest()
	}
}

func (c *L1Cache) removeOldest() {
	ent := c.evictList.Back()
	if ent != nil {
		c.removeElement(ent)
	}
}

func (c *L1Cache) removeElement(e *list.Element) {
	c.evictList.Remove(e)
	item := e.Value.(*l1Item)
	delete(c.items, item.key)
	c.currBytes -= item.size
}

// Remove explicitly removes an entry from the L1 cache if it exists.
func (c *L1Cache) Remove(tenantID, query string) {
	key := tenantID + ":" + query
	c.mu.Lock()
	defer c.mu.Unlock()

	if ent, ok := c.items[key]; ok {
		c.removeElement(ent)
	}
}
