package bcache

import (
	"encoding/binary"
	"github.com/valyala/bytebufferpool"
	"sync"
	"time"
)

type Options struct {
	maxBytes uint64
	h        Hash
}

type Option func(opt *Options)

func MaxBytes(n uint64) Option {
	return func(opt *Options) {
		opt.maxBytes = n
	}
}

func WithHash(h Hash) Option {
	return func(opt *Options) {
		opt.h = h
	}
}

func New(options ...Option) (cache *Cache) {
	opt := Options{}
	for _, option := range options {
		option(&opt)
	}
	if opt.maxBytes < 1 {
		opt.maxBytes = defaultMaxBytes
	} else if opt.maxBytes >= maxBucketSize {
		opt.maxBytes = maxBucketSize - 1<<30
	}
	if opt.h == nil {
		opt.h = MemHash{}
	}
	cache = &Cache{
		locker:       sync.RWMutex{},
		maxItemBytes: opt.maxBytes / 2,
		buckets:      [bucketsCount]bucket{},
		hash:         opt.h,
	}
	maxBucketBytes := (opt.maxBytes + bucketsCount - 1) / bucketsCount
	for i := range cache.buckets[:] {
		cache.buckets[i].create(maxBucketBytes, cache.evict)
	}
	return
}

type Cache struct {
	locker       sync.RWMutex
	maxItemBytes uint64
	buckets      [bucketsCount]bucket
	hash         Hash
}

func (c *Cache) canSet(k []byte, v []byte) (ok bool) {
	vLen := len(v)
	if vLen == 0 {
		vLen = 8
	}
	itemLen := uint64(len(k) + vLen + 4 + 10)
	ok = itemLen < c.maxItemBytes
	return
}

func (c *Cache) set(k []byte, v []byte, h uint64) {
	idx := h % bucketsCount
	c.buckets[idx].Set(k, v, h)
}

func (c *Cache) get(k []byte) (p []byte, found bool) {
	p = make([]byte, 0, 8)
	h := c.hash.Sum(k)
	idx := h % bucketsCount
	p, found = c.buckets[idx].Get(p, k, h, true)
	return
}

func (c *Cache) contains(k []byte) (ok bool) {
	h := c.hash.Sum(k)
	idx := h % bucketsCount
	_, ok = c.buckets[idx].Get(nil, k, h, false)
	return
}

func (c *Cache) SetWithTTL(k []byte, v []byte, ttl time.Duration) (err error) {
	if len(k) == 0 || len(v) == 0 {
		err = ErrInvalidKey
		return
	}

	if !c.canSet(k, v) {
		err = ErrTooBigKey
		return
	}
	c.locker.Lock()
	kvs := MakeEntries(k, v, ttl, c.hash)
	for _, kv := range kvs {
		c.set(kv.k, kv.v, kv.h)
	}
	c.locker.Unlock()
	return
}

func (c *Cache) Set(k []byte, v []byte) (err error) {
	err = c.SetWithTTL(k, v, 0)
	return
}

func (c *Cache) Get(k []byte) (p []byte, ok bool) {
	c.locker.RLock()
	// first
	dst, found := c.get(k)
	if !found {
		c.locker.RUnlock()
		return
	}
	v := Value(dst)
	if v.Pos() > 1 {
		c.locker.RUnlock()
		return
	}
	if deadline := v.Deadline(); !deadline.IsZero() {
		if deadline.Before(time.Now()) {
			c.locker.RUnlock()
			return
		}
	}
	size := v.Size()
	if size == 1 {
		p = v.Bytes()
		ok = true
		c.locker.RUnlock()
		return
	}

	// big key
	kLen := len(k)
	nkLen := kLen + 8
	b := bytebufferpool.Get()
	_, _ = b.Write(v.Bytes())
	for i := 2; i <= size; i++ {
		nk := make([]byte, nkLen)
		copy(nk, k)
		binary.BigEndian.PutUint64(nk[kLen:], uint64(i))
		np, has := c.get(nk)
		if !has {
			return
		}
		_, _ = b.Write(Value(np).Bytes())
	}
	p = b.Bytes()
	bytebufferpool.Put(b)
	ok = true
	c.locker.RUnlock()
	return
}

func (c *Cache) Contains(k []byte) bool {
	c.locker.RLock()
	defer c.locker.RUnlock()
	return c.contains(k)
}

func (c *Cache) Expire(k []byte, ttl time.Duration) {
	c.locker.Lock()
	dst, found := c.get(k)
	if !found {
		c.locker.Unlock()
		return
	}
	v := Value(dst)
	if v.Pos() > 1 {
		c.locker.Unlock()
		return
	}
	v.SetDeadline(time.Now().Add(ttl))
	c.locker.Unlock()
	return
}

func (c *Cache) Remove(k []byte) {
	if len(k) > maxKeyLen {
		return
	}
	c.locker.Lock()
	h := c.hash.Sum(k)
	idx := h % bucketsCount
	c.buckets[idx].Remove(h)
	c.locker.Unlock()
}

func (c *Cache) evict(_ uint64) {

}
