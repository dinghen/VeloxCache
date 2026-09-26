# Quality Guidelines

> Code quality standards for backend development.

---

## Overview

<!--
Document your project's quality standards here.

Questions to answer:
- What patterns are forbidden?
- What linting rules do you enforce?
- What are your testing requirements?
- What code review standards apply?
-->

(To be filled by the team)

## Required Patterns

- Use `sync.Map.LoadOrStore` for singleflight ownership decisions.
- Do not mutate maps or linked-list state while holding an `RLock`.
- Do not invoke user callbacks while holding a storage lock.
- Add a regression test for every fixed concurrency or lifecycle bug.
- Run `gofmt`, `go vet ./...`, `go test ./...`, and `go test -race ./...` before commit.

## Forbidden Patterns

- Do not use a second `Load`/`Store` pair where an atomic ownership decision is required.
- Do not use context values as a peer-routing protocol.
- Do not use a fabricated long TTL to represent a non-expiring entry.
- Do not close a channel or resource without an idempotence guard.

## Testing Requirements

Cross-layer routing changes require unit tests for the picker boundary and at least one fake-owner Get/Set/Delete test. Storage changes require TTL, promotion, deletion, clear, and close coverage.

---

## Forbidden Patterns

<!-- Patterns that should never be used and why -->

(To be filled by the team)

---

## Required Patterns

<!-- Patterns that must always be used -->

(To be filled by the team)

---

## Testing Requirements

<!-- What level of testing is expected -->

(To be filled by the team)

---

## Code Review Checklist

<!-- What reviewers should check -->

(To be filled by the team)
