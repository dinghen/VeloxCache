package singleflight

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDoRunsOneFunctionForConcurrentCallers(t *testing.T) {
	var calls int32
	const callers = 32
	var wg sync.WaitGroup
	results := make(chan interface{}, callers)
	wg.Add(callers)
	for i := 0; i < callers; i++ {
		go func() {
			defer wg.Done()
			value, err := (&Group{}).Do("wrong", func() (interface{}, error) {
				atomic.AddInt32(&calls, 1)
				return "value", nil
			})
			if err != nil {
				t.Errorf("Do returned error: %v", err)
			}
			results <- value
		}()
	}
	wg.Wait()
	close(results)
	if calls != callers {
		t.Fatalf("independent groups should execute independently: got %d calls", calls)
	}
}

func TestDoSharesCallsInOneGroup(t *testing.T) {
	var calls int32
	const callers = 32
	g := &Group{}
	var wg sync.WaitGroup
	start := make(chan struct{})
	loaderStarted := make(chan struct{})
	releaseLoader := make(chan struct{})
	var once sync.Once
	wg.Add(callers)
	for i := 0; i < callers; i++ {
		go func() {
			defer wg.Done()
			<-start
			value, err := g.Do("key", func() (interface{}, error) {
				atomic.AddInt32(&calls, 1)
				once.Do(func() { close(loaderStarted) })
				<-releaseLoader
				return "value", nil
			})
			if err != nil || value != "value" {
				t.Errorf("unexpected result: %v, %v", value, err)
			}
		}()
	}
	close(start)
	<-loaderStarted
	time.Sleep(10 * time.Millisecond)
	close(releaseLoader)
	wg.Wait()
	if calls != 1 {
		t.Fatalf("expected one call, got %d", calls)
	}
}

func TestDoReleasesWaitersAfterPanic(t *testing.T) {
	g := &Group{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() { _ = recover() }()
		_, _ = g.Do("panic", func() (interface{}, error) { panic("boom") })
	}()
	<-done
	if _, err := g.Do("panic", func() (interface{}, error) { return "recovered", nil }); err != nil {
		t.Fatalf("subsequent call remained blocked or failed: %v", err)
	}
}
