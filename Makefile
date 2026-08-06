CLI_APP := steam-profit
SERVER_APP := steam-profit-server
CLI_PACKAGE := ./cmd/steam-profit
SERVER_PACKAGE := ./cmd/steam-profit-server
BUILD_DIR := .build
GO_LDFLAGS :=
POST_BUILD := true

ifeq ($(shell uname -s),Darwin)
GO_LDFLAGS := -linkmode=external
POST_BUILD := codesign --force --sign -
endif

.PHONY: build check clean fmt fmt-check run-web test test-race vet

build:
	mkdir -p $(BUILD_DIR)
	go build -trimpath -ldflags="$(GO_LDFLAGS)" -o $(BUILD_DIR)/$(CLI_APP) $(CLI_PACKAGE)
	$(POST_BUILD) $(BUILD_DIR)/$(CLI_APP)
	go build -trimpath -ldflags="$(GO_LDFLAGS)" -o $(BUILD_DIR)/$(SERVER_APP) $(SERVER_PACKAGE)
	$(POST_BUILD) $(BUILD_DIR)/$(SERVER_APP)

run-web:
	go run $(SERVER_PACKAGE)

fmt:
	gofmt -w cmd internal

fmt-check:
	test -z "$(shell gofmt -l cmd internal)"

vet:
	go vet ./...

test:
	go test -ldflags="$(GO_LDFLAGS)" ./...

test-race:
	go test -race -ldflags="$(GO_LDFLAGS)" ./...

check: fmt-check vet test-race build

clean:
	rm -rf $(BUILD_DIR)
