# bcache
byte cache

## Install
```shell
go get github.com/aacfactory/bcache
```

## Usage
```go
cache := bcache.New()
setErr := cache.Set([]byte("key"), []byte("value"))
value, has := cache.Get([]byte("key"))
```