package veloxcache

import (
	"context"
	"testing"
)

type routingPeerPicker struct {
	peer Peer
	self bool
}

func (p routingPeerPicker) PickPeer(string) (Peer, bool, bool) { return p.peer, true, p.self }
func (p routingPeerPicker) Close() error                       { return nil }

type routingPeer struct {
	setCalls, getCalls, deleteCalls int
	value                           []byte
}

func (p *routingPeer) Get(string, string) ([]byte, error) {
	p.getCalls++
	return append([]byte(nil), p.value...), nil
}
func (p *routingPeer) Set(_ context.Context, _, _ string, value []byte) error {
	p.setCalls++
	p.value = append([]byte(nil), value...)
	return nil
}
func (p *routingPeer) Delete(string, string) (bool, error) {
	p.deleteCalls++
	p.value = nil
	return true, nil
}
func (p *routingPeer) Close() error { return nil }

type groupPeer struct{ target *Group }

func (p groupPeer) Get(_ string, key string) ([]byte, error) {
	value, err := p.target.getLocal(context.Background(), key)
	if err != nil {
		return nil, err
	}
	return value.ByteSLice(), nil
}
func (p groupPeer) Set(ctx context.Context, _, key string, value []byte) error {
	return p.target.setLocal(key, value)
}
func (p groupPeer) Delete(_ string, key string) (bool, error) {
	if err := p.target.deleteLocal(key); err != nil {
		return false, err
	}
	return true, nil
}
func (p groupPeer) Close() error { return nil }

func TestGroupRoutesAllOperationsToOwner(t *testing.T) {
	peer := &routingPeer{}
	g := NewGroup("routing-group", 1024, GetterFunc(func(context.Context, string) ([]byte, error) {
		t.Fatal("remote owner should prevent local getter")
		return nil, nil
	}), WithPeers(routingPeerPicker{peer: peer}))
	defer g.Close()

	if err := g.Set(context.Background(), "key", []byte("value")); err != nil {
		t.Fatal(err)
	}
	if _, err := g.Get(context.Background(), "key"); err != nil {
		t.Fatal(err)
	}
	if err := g.Delete(context.Background(), "key"); err != nil {
		t.Fatal(err)
	}
	if peer.setCalls != 1 || peer.getCalls != 1 || peer.deleteCalls != 1 {
		t.Fatalf("unexpected owner calls: %+v", peer)
	}
}

func TestTwoGroupsSingleOwnerFlow(t *testing.T) {
	getter := GetterFunc(func(context.Context, string) ([]byte, error) {
		return []byte("loaded"), nil
	})
	owner := NewGroup("two-node-owner", 1024, getter)
	client := NewGroup("two-node-client", 1024, getter)
	defer owner.Close()
	defer client.Close()
	client.RegisterPeers(routingPeerPicker{peer: groupPeer{target: owner}})

	if err := client.Set(context.Background(), "key", []byte("value")); err != nil {
		t.Fatal(err)
	}
	value, err := client.Get(context.Background(), "key")
	if err != nil || value.String() != "value" {
		t.Fatalf("remote get failed: %q, %v", value.String(), err)
	}
	if err := client.Delete(context.Background(), "key"); err != nil {
		t.Fatal(err)
	}
	if _, found := owner.mainCache.Get(context.Background(), "key"); found {
		t.Fatal("owner retained deleted key")
	}
}
