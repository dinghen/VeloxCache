package veloxcache

import (
	"context"
	"sync"
	"testing"
)

func TestCacheConcurrentCloseAndWrites(t *testing.T) {
	c := NewCache(DefaultCacheOptions())
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				c.Add("key", ByteView{b: []byte("value")})
				c.Get(nil, "key")
			}
		}(i)
	}
	c.Close()
	c.Close()
	wg.Wait()
}

func TestDestroyGroupDoesNotDeadlock(t *testing.T) {
	name := "destroy-group-test"
	g := NewGroup(name, 1024, GetterFunc(func(_ context.Context, _ string) ([]byte, error) {
		return []byte("value"), nil
	}))
	if !DestroyGroup(name) || GetGroup(name) != nil {
		t.Fatal("group was not destroyed")
	}
	_ = g.Close()
}
