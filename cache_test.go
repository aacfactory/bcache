package bcache_test

import (
	"bytes"
	"fmt"
	"github.com/aacfactory/bcache"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	cache := bcache.New()
	keyA := []byte("a")
	setKeyAErr := cache.Set(keyA, keyA)
	if setKeyAErr != nil {
		t.Error(setKeyAErr)
		return
	}
	valA, hasA := cache.Get(keyA)
	fmt.Println("a:", hasA, string(valA))
	// big
	keyB := []byte("b")
	big := [2 << 16]byte{}
	copy(big[0:1], []byte{'b'})
	copy(big[len(big)-1:], []byte{'b'})
	setKeyBErr := cache.Set(keyB, big[:])
	if setKeyBErr != nil {
		t.Error(setKeyBErr)
		return
	}
	valB, hasB := cache.Get(keyB)
	fmt.Println("b:", hasB, len(valB), len(valB) == len(big), bytes.Equal(big[:], valB))

	// ttl
	keyC := []byte("c")
	setKeyCErr := cache.SetWithTTL(keyC, keyC, 1*time.Second)
	if setKeyCErr != nil {
		t.Error(setKeyCErr)
		return
	}
	valC, hasC := cache.Get(keyC)
	fmt.Println("c:", hasC, string(valC))
	time.Sleep(1 * time.Second)
	valC, hasC = cache.Get(keyC)
	fmt.Println("c:", hasC, string(valC))
	cache.Remove(keyB)
}
