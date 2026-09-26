# Implementation Plan

1. Add characterization tests for single-owner routing, peer identity, lifecycle, and the current LRU2 failures.
2. Implement local-only Group operations and route Get/Set/Delete to the consistent-hash owner.
3. Add the local node to the picker ring, normalize addresses, and correct initial/watch event handling needed by the routing contract.
4. Fix server/registry option copying, stop signaling, lease cleanup, and idempotent shutdown.
5. Fix singleflight atomic registration and panic-safe cleanup.
6. Fix consistent-hash locking and make the adaptive rebalancer safe or disable it until its behavior is deterministic.
7. Fix Cache/LRU/LRU2 lifecycle and locking defects while preserving public APIs.
8. Make LRU2 tests green and add focused tests for TTL, promotion, deletion, clear, and close.
9. Run `gofmt`, `go vet ./...`, `go test ./...`, and targeted concurrency tests; inspect for goroutine leaks and request loops.

Rollback points:

- Routing changes are isolated behind local-only Group helpers and can be reverted without changing the protobuf contract.
- Storage changes are limited to existing Store implementations and can be reverted independently of routing.
