CLI_APP := steam-profit
SERVER_APP := steam-profit-server
CLI_PACKAGE := ./cmd/steam-profit
SERVER_PACKAGE := ./cmd/steam-profit-server
BUILD_DIR := .build
GO_LDFLAGS :=
POST_BUILD := true
PNPM ?= pnpm
DOCKER ?= docker

ifeq ($(shell uname -s),Darwin)
GO_LDFLAGS := -linkmode=external
POST_BUILD := codesign --force --sign -
endif

.PHONY: build check clean container-check demo-run e2e e2e-install fmt fmt-check js-check run-web test test-race vet

build:
	mkdir -p $(BUILD_DIR)
	go build -trimpath -ldflags="$(GO_LDFLAGS)" -o $(BUILD_DIR)/$(CLI_APP) $(CLI_PACKAGE)
	$(POST_BUILD) $(BUILD_DIR)/$(CLI_APP)
	go build -trimpath -ldflags="$(GO_LDFLAGS)" -o $(BUILD_DIR)/$(SERVER_APP) $(SERVER_PACKAGE)
	$(POST_BUILD) $(BUILD_DIR)/$(SERVER_APP)

run-web:
	go run $(SERVER_PACKAGE)

demo-run: build
	mkdir -p $(BUILD_DIR)/demo
	$(BUILD_DIR)/$(SERVER_APP) --addr 127.0.0.1:8080 --db $(BUILD_DIR)/demo/steam-profit.db --import-csv demo/seed.csv

e2e-install:
	$(PNPM) install --frozen-lockfile
	$(PNPM) exec playwright install chromium

e2e:
	$(PNPM) e2e

container-check:
	$(DOCKER) build --tag steam-profit-calculator:check .

fmt:
	gofmt -w cmd internal

fmt-check:
	test -z "$(shell gofmt -l cmd internal)"

vet:
	go vet ./...

js-check:
	node --check internal/webapi/web/app.js

test:
	go test -ldflags="$(GO_LDFLAGS)" ./...

test-race:
	go test -race -ldflags="$(GO_LDFLAGS)" ./...

check: fmt-check js-check vet test-race build

clean:
	rm -rf $(BUILD_DIR)
