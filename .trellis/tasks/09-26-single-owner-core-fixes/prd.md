# Implement single-owner routing and core bug fixes

## Goal

Stabilize the cache core and distributed routing without adding new product features.

## Background

The repository currently mixes local caching, consistent-hash routing, and asynchronous peer replication. That produces incorrect Set/Delete behavior, self-peer loops, concurrency hazards, lifecycle leaks, and failing LRU2 tests.

## Requirements

1. Adopt single-owner routing: the consistent-hash ring includes the local node; Get, Set, and Delete route a key to its owner, and the owner executes the operation locally.
2. Remove the existing asynchronous replication path and the `from_peer` context convention from the request flow.
3. Normalize advertised and local peer addresses so a node can reliably identify itself.
4. Fix confirmed correctness defects in singleflight, consistent hash, cache lifecycle, server registration/stop, group destruction, service-discovery event handling, LRU, and LRU2 without adding new product features.
5. Preserve existing public APIs where practical. Keep LRU2 as the storage implementation and make the smallest internal fixes required for its documented behavior.
6. Add regression and integration coverage for the changed behavior.

## Constraints

- Do not add persistence, replication, batching, metrics, limits, new cache policies, or a new error/status API.
- Do not introduce new runtime dependencies.
- Keep changes within the existing Go package layout.

## Acceptance Criteria

- [x] `go test ./...` passes, including existing LRU2 tests.
- [x] Concurrent singleflight calls execute the loader once and do not block forever after a loader panic.
- [x] Concurrent consistent-hash lookups do not write maps under read locks or deadlock during rebalancing.
- [x] Two-node Get/Set/Delete tests demonstrate one owner per key and no peer-to-peer request loop.
- [x] A node can stop without leaving its registration lease running or panicking on repeated Stop.
- [x] Group destruction and cache close are safe under repeated/concurrent calls.
- [x] LRU2 supports promotion, expiration, non-expiring entries, deletion, clearing, and clean shutdown consistently with its Store contract.
- [x] `go vet ./...` passes.
