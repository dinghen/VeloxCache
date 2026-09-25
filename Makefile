GO ?= go

# Keep build and module caches in a writable, project-specific temporary path.
CACHE_ROOT ?= /tmp/veloxcache-go
GOCACHE ?= $(CACHE_ROOT)/build
GOMODCACHE ?= $(CACHE_ROOT)/mod
GOPATH ?= $(CACHE_ROOT)/path
export GOCACHE GOMODCACHE GOPATH

PORT ?= 8001
NODE ?= A

GO_MODULES := \
	github.com/sirupsen/logrus@v1.9.3 \
	google.golang.org/grpc@v1.70.0 \
	google.golang.org/protobuf@v1.36.4 \
	go.etcd.io/etcd/client/v3@v3.5.18

.PHONY: setup test vet run-example etcd-up etcd-down

setup:
	@mkdir -p "$(GOCACHE)" "$(GOMODCACHE)" "$(GOPATH)"
	$(GO) mod download $(GO_MODULES)

test: setup
	$(GO) test ./...

vet: setup
	$(GO) vet ./...

run-example: setup
	$(GO) run ./example -port "$(PORT)" -node "$(NODE)"

etcd-up:
	docker compose up -d etcd

etcd-down:
	docker compose down
