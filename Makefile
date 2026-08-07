CLI_APP := steam-profit
SERVER_APP := steam-profit-server
EVAL_APP := steam-profit-eval
CLI_PACKAGE := ./cmd/steam-profit
SERVER_PACKAGE := ./cmd/steam-profit-server
EVAL_PACKAGE := ./cmd/steam-profit-eval
BUILD_DIR := .build
GO_LDFLAGS :=
POST_BUILD := true
PNPM ?= pnpm
DOCKER ?= docker

ifeq ($(shell uname -s),Darwin)
GO_LDFLAGS := -linkmode=external
POST_BUILD := codesign --force --sign -
endif

.PHONY: ai-eval ai-eval-baseline build check clean container-check demo-run e2e e2e-install fmt fmt-check js-check run-web test test-race vet

build:
	mkdir -p $(BUILD_DIR)
	go build -trimpath -ldflags="$(GO_LDFLAGS)" -o $(BUILD_DIR)/$(CLI_APP) $(CLI_PACKAGE)
	$(POST_BUILD) $(BUILD_DIR)/$(CLI_APP)
	go build -trimpath -ldflags="$(GO_LDFLAGS)" -o $(BUILD_DIR)/$(SERVER_APP) $(SERVER_PACKAGE)
	$(POST_BUILD) $(BUILD_DIR)/$(SERVER_APP)
	go build -trimpath -ldflags="$(GO_LDFLAGS)" -o $(BUILD_DIR)/$(EVAL_APP) $(EVAL_PACKAGE)
	$(POST_BUILD) $(BUILD_DIR)/$(EVAL_APP)

run-web:
	go run $(SERVER_PACKAGE)

demo-run: build
	mkdir -p $(BUILD_DIR)/demo
	$(BUILD_DIR)/$(SERVER_APP) --addr 127.0.0.1:8080 --db $(BUILD_DIR)/demo/steam-profit.db --import-csv demo/seed.csv

ai-eval:
	go run $(EVAL_PACKAGE) --live --prompt-version current

ai-eval-baseline:
	go run $(EVAL_PACKAGE) --live --prompt-version baseline --output $(BUILD_DIR)/evaluations/baseline-report.json

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
