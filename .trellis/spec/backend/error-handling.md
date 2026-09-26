# Error Handling

> How errors are handled in this project.

---

## Overview

<!--
Document your project's error handling conventions here.

Questions to answer:
- What error types do you define?
- How are errors propagated?
- How are errors logged?
- How are errors returned to clients?
-->

(To be filled by the team)

## Cache and Routing Contracts

### 1. Scope / Trigger

These contracts apply to Group routing, peer RPCs, cache lifecycle, and storage operations.

### 2. Signatures

- `Group.Get(ctx context.Context, key string) (ByteView, error)`
- `Group.Set(ctx context.Context, key string, value []byte) error`
- `Group.Delete(ctx context.Context, key string) error`
- `PeerPicker.PickPeer(key string) (Peer, bool, bool)`
- `Group.getLocal`, `Group.setLocal`, and `Group.deleteLocal` are internal RPC targets.

### 3. Contracts

- `PickPeer` selects exactly one owner. The local owner returns `self=true`; a remote owner returns a `Peer`.
- Public Group methods route to the selected owner before reading or mutating local storage.
- Server RPC handlers call the local-only helpers and must not route again.
- Empty keys return `ErrKeyRequired`; empty values return `ErrValueRequired` for Set; unavailable ownership returns `ErrNoOwner`.
- Remote errors are wrapped with `%w` so callers can inspect the underlying error.

### 4. Validation & Error Matrix

| Condition | Result |
|---|---|
| Group closed | `ErrGroupClosed` |
| Empty key | `ErrKeyRequired` |
| Empty Set value | `ErrValueRequired` |
| Picker has no owner | `ErrNoOwner` |
| Remote RPC fails | wrapped error from the owner call |

### 5. Good/Base/Bad Cases

- Good: one key selects one owner and a remote request reaches that owner exactly once.
- Base: no picker is configured; Group uses its local cache and getter.
- Bad: a server RPC calls `Group.Get`/`Set`/`Delete` and routes a request again.

### 6. Tests Required

- Route Get, Set, and Delete to a fake remote owner and assert one call per operation.
- Assert local-only server handling does not invoke the picker.
- Assert empty input and closed groups return the sentinel errors.

### 7. Wrong vs Correct

#### Wrong

```go
// A server-side peer request routes again and can loop between nodes.
return group.Set(ctx, req.Key, req.Value)
```

#### Correct

```go
// RPC is already at the owner; execute local storage directly.
return group.setLocal(req.Key, req.Value)
```

## Lifecycle Rules

- Copy default server options before applying per-instance options.
- Stop signals and cache/store Close methods are idempotent.
- Background watchers and cleanup loops must select on an owned close channel.
- `DestroyGroup` removes the registry entry before closing the group; never acquire `groupsMu` recursively.

---

## Error Types

<!-- Custom error classes/types -->

(To be filled by the team)

---

## Error Handling Patterns

<!-- Try-catch patterns, error propagation -->

(To be filled by the team)

---

## API Error Responses

<!-- Standard error response format -->

(To be filled by the team)

---

## Common Mistakes

<!-- Error handling mistakes your team has made -->

(To be filled by the team)
