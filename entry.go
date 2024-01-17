package bcache

import (
	"encoding/binary"
	"github.com/valyala/bytebufferpool"
	"time"
)

func MakeEntries(k []byte, p []byte, ttl time.Duration, hash Hash) (entries Entries) {
	entries = make([]Entry, 0, 1)
	kLen := len(k)
	pLen := len(p)
	if 4+kLen+10+pLen < chunkSize {
		//normal
		v := make([]byte, 10+pLen)
		v[0] = 1
		v[1] = 1
		if ttl > 0 {
			binary.BigEndian.PutUint64(v[2:10], uint64(time.Now().Add(ttl).UnixNano()))
		} else {
			binary.BigEndian.PutUint64(v[2:10], 0)
		}
		copy(v[10:], p)
		entries = append(entries, Entry{
			k: k,
			v: v,
			h: hash.Sum(k),
		})
		return
	}
	// big key
	// first
	p0Len := chunkSize - 4 - kLen - 10 - 1
	p0 := p[0:p0Len]
	v0 := make([]byte, 10+p0Len)
	v0[0] = 1
	if ttl > 0 {
		binary.BigEndian.PutUint64(v0[2:10], uint64(time.Now().Add(ttl).UnixNano()))
	} else {
		binary.BigEndian.PutUint64(v0[2:10], 0)
	}
	copy(v0[10:], p0)
	entries = append(entries, Entry{
		k: k,
		v: v0,
		h: hash.Sum(k),
	})
	// next
	p = p[p0Len:]
	maxChunkValueLen := chunkSize - 4 - kLen - 10 - 1
	pos := uint64(2)
	stop := false
	for {
		chunkValueLen := 0
		if npLen := len(p); npLen <= maxChunkValueLen {
			chunkValueLen = npLen
			stop = true
		} else {
			chunkValueLen = maxChunkValueLen
		}
		nk := make([]byte, kLen+8)
		copy(nk, k)
		binary.BigEndian.PutUint64(nk[kLen:], pos)

		np := make([]byte, 2+chunkValueLen)
		np[0] = byte(pos)
		copy(np[2:], p[0:chunkValueLen])
		entries = append(entries, Entry{
			k: nk,
			v: np,
			h: hash.Sum(nk),
		})
		if stop {
			break
		}
		pos++
		p = p[chunkValueLen:]
	}

	entriesLen := len(entries)
	for _, entry := range entries {
		entry.v[1] = byte(entriesLen)
	}
	return
}

// Value
// [1]pos [1]size, [8]deadline, [...]value
type Value []byte

func (v Value) Pos() int {
	return int(v[0])
}

func (v Value) Size() int {
	return int(v[1])
}

func (v Value) Normal() bool {
	return v.Size() == 1
}

func (v Value) BigKey() bool {
	return v.Size() > 1
}

func (v Value) SetDeadline(deadline time.Time) {
	if v.Pos() == 1 {
		n := deadline.UnixNano()
		binary.BigEndian.PutUint64(v[2:10], uint64(n))
	}
}

func (v Value) Deadline() time.Time {
	if v.Pos() == 1 {
		n := binary.BigEndian.Uint64(v[2:10])
		if n == 0 {
			return time.Time{}
		}
		return time.Unix(0, int64(n))
	}
	return time.Time{}
}

func (v Value) Bytes() (p []byte) {
	if v.Pos() == 1 {
		p = v[10:]
	} else {
		p = v[2:]
	}
	return
}

type Entry struct {
	k []byte
	v Value
	h uint64
}

func (kv Entry) Key() []byte {
	return kv.k
}

func (kv Entry) Value() Value {
	return kv.v
}

func (kv Entry) Hash() uint64 {
	return kv.h
}

type Entries []Entry

func (entries Entries) Value() (p []byte) {
	b := bytebufferpool.Get()
	for _, entry := range entries {
		_, _ = b.Write(entry.v.Bytes())
	}
	p = b.Bytes()
	bytebufferpool.Put(b)
	return
}

func (entries Entries) Deadline() time.Time {
	return entries[0].v.Deadline()
}
