.PHONY: run-all generate tools test

GO ?= go
PROTOC ?= protoc
export PATH := $(CURDIR)/bin:$(PATH)

tools:
	GOBIN=$(CURDIR)/bin $(GO) install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.12
	GOBIN=$(CURDIR)/bin $(GO) install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.1

generate: tools
	$(PROTOC) --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative api/product/v1/product.proto

run-all:
	@mkdir -p bin
	$(GO) build -o bin/product-service ./cmd/product-service
	$(GO) build -o bin/cart ./cmd/cart
	@set -e; ./bin/product-service & product_pid=$$!; \
	./bin/cart & cart_pid=$$!; \
	trap 'kill $$product_pid $$cart_pid 2>/dev/null || true' EXIT; \
	trap 'exit 0' INT TERM; \
	while kill -0 $$product_pid 2>/dev/null && kill -0 $$cart_pid 2>/dev/null; do sleep 1; done; \
	exit 1

test:
	$(GO) test -race ./...
