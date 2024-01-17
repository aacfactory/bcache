package bcache

import "github.com/aacfactory/bcache/mmhash"

type Hash interface {
	Sum(p []byte) (h uint64)
}

type MemHash struct{}

func (hash MemHash) Sum(p []byte) (h uint64) {
	h = mmhash.Sum64(p)
	return
}
