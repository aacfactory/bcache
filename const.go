package bcache

import "fmt"

const (
	defaultMaxBytes        = 64 << (10 * 2)
	bucketsCount           = 512
	chunkSize              = 1 << 16
	bucketSizeBits         = 40
	genSizeBits            = 64 - bucketSizeBits
	maxGen                 = 1<<genSizeBits - 1
	maxBucketSize   uint64 = 1 << bucketSizeBits
	maxKeyLen              = chunkSize - 4 - 2 - 8 - 8
)

var (
	ErrTooBigKey  = fmt.Errorf("key was too big, must not be greater than 63k")
	ErrInvalidKey = fmt.Errorf("key is invalid")
)
