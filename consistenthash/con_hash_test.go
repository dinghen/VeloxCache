package consistenthash

import (
	"sync"
	"testing"
)

func TestConcurrentGetAndMembershipChanges(t *testing.T) {
	m := New()
	defer m.Close()
	if err := m.Add("a", "b", "c"); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				_ = m.Get("key")
				if id == 0 && j%100 == 0 {
					_ = m.GetStats()
				}
			}
		}(i)
	}
	wg.Wait()
	if got := m.Get("key"); got == "" {
		t.Fatal("expected a node for a non-empty ring")
	}
}
