package singleflight

import (
	"fmt"
	"sync"
)

// 代表正在进行或已结束的请求
type call struct {
	wg  sync.WaitGroup
	val interface{}
	err error
}

// Group manages all kinds of calls
type Group struct {
	m sync.Map // 使用sync.Map来优化并发性能
}

// Do 针对相同的key，保证多次调用Do()，都只会调用一次fn
func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	c := &call{}
	c.wg.Add(1)

	// LoadOrStore is the ownership decision. A Load followed by Store allows
	// two callers to execute fn concurrently for the same key.
	actual, loaded := g.m.LoadOrStore(key, c)
	if loaded {
		existing := actual.(*call)
		existing.wg.Wait()
		return existing.val, existing.err
	}

	// Always release waiters and remove the entry, including when fn panics.
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				c.err = fmt.Errorf("singleflight function panicked: %v", recovered)
				c.wg.Done()
				g.m.Delete(key)
				panic(recovered)
			}
		}()
		c.val, c.err = fn()
		c.wg.Done()
		g.m.Delete(key)
	}()

	return c.val, c.err
}
