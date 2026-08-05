APP := steam-profit
PACKAGE := ./cmd/steam-profit
BUILD_DIR := .build
GO_LDFLAGS :=
POST_BUILD := true

ifeq ($(shell uname -s),Darwin)
GO_LDFLAGS := -linkmode=external
POST_BUILD := codesign --force --sign -
endif

.PHONY: build check clean fmt fmt-check test test-race vet

build:
	mkdir -p $(BUILD_DIR)
	go build -trimpath -ldflags="$(GO_LDFLAGS)" -o $(BUILD_DIR)/$(APP) $(PACKAGE)
	$(POST_BUILD) $(BUILD_DIR)/$(APP)

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
