# Alex Mackay 2026


# Build folder (CLI)
BUILD_FOLDER = build
COVERAGE_BUILD_FOLDER    ?= $(BUILD_FOLDER)/coverage
UNIT_COVERAGE_OUT        ?= $(COVERAGE_BUILD_FOLDER)/ut_cov.out
BIN                      ?= $(BUILD_FOLDER)/agent-cli

# Packages
PKG                      ?= github.com/ATMackay/agent-cli
CONSTANTS_PKG            ?= $(PKG)/constants


# Git based version
VERSION_TAG    ?= $(shell git describe --tags)
GIT_COMMIT     ?= $(shell git rev-parse HEAD)
BUILD_DATE     ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
COMMIT_DATE    ?= $(shell TZ=UTC git show -s --format=%cd --date=format:%Y-%m-%dT%H:%M:%SZ HEAD)
ifndef DIRTY
DIRTY := $(shell if [ -n "$$(git status --porcelain 2>/dev/null)" ]; then echo true; else echo false; fi)
endif

build:
	@mkdir -p build
	@echo ">> building $(BIN) (version=$(VERSION_TAG) commit=$(GIT_COMMIT) dirty=$(DIRTY))"
	GO111MODULE=on go build -ldflags "$(LDFLAGS)" -o $(BIN)
	@echo  "Agent server successfully built. To run the application execute './$(BIN) run'"

install: build
	mv $(BIN) $(GOBIN)

run: build
	@./$(BUILD_FOLDER)/agent-cli run documentor --repo https://github.com/ATMackay/agent.git

test: 
	@mkdir -p $(COVERAGE_BUILD_FOLDER)
	@go test -cover -coverprofile $(UNIT_COVERAGE_OUT) -v ./...

.PHONY: build install run test