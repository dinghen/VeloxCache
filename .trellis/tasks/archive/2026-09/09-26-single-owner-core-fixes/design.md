# Technical Design

## Change Boundary

The behavior gap is between the current mixed local/replicated request flow and a single-owner cache: routing currently happens only after a local miss, writes are copied asynchronously, and peer identity is not stable. The routing boundary lives in `Group` and `ClientPicker`; local storage remains in `Cache` and `store`.

Expected files to change:

- `group.go`, `peers.go`, `server.go`, `client.go`, and `registry/register.go`: owner routing, local RPC execution, address and lifecycle fixes.
- `consistenthash/con_hash.go`: synchronization and rebalancer correctness.
- `singleflight/singleflight.go`: atomic call registration and panic-safe cleanup.
- `cache.go`, `store/lru.go`, `store/lru2.go`: lifecycle, locking, and existing storage behavior fixes.
- Existing and new Go tests next to the affected packages.

## Routing Contract

`PeerPicker` represents every node in the ring, including the local node. `PickPeer` returns `self=true` for the local owner and a `Peer` for a remote owner. `Group` routes before consulting its cache:

1. Pick the owner.
2. If the owner is remote, call the peer RPC once.
3. If the owner is local, execute a local-only cache/load operation.

Server RPC methods call the local-only operation directly, so a remote request cannot re-enter routing. The old `from_peer` context marker and asynchronous `syncToPeers` path are removed from the active flow.

## Lifecycle and Concurrency

- Copy default options per server instance.
- Use an owned stop signal and idempotent shutdown for the server and registry.
- Use `sync.Map.LoadOrStore` plus deferred completion/cleanup in singleflight.
- Protect map writes with exclusive locks; do not mutate state under `RLock`.
- Make cache/store Close idempotent and ensure operations cannot observe a nil store after acquiring their operation lock.
- Never invoke user eviction callbacks while holding an internal cache lock.

## LRU2 Scope

Keep the existing two-level structure. Fix promotion, expiration zero semantics, duplicate residency, eviction callbacks, and cleanup goroutine shutdown. Do not replace it with a different policy.

## Explicitly Out Of Scope

Full etcd watch resynchronization/backoff policy, TLS client support, gRPC status-code redesign, new observability, limits, persistence, replication, and performance tuning beyond correctness fixes.

