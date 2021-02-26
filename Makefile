NAME = kitchen
GOENV = CGO_ENABLED=0
GO = $(GOENV) go

all: mod tools-all gen test-all build
.PHONY: all

mod:
	$(GO) mod download
mod-tidy:
	$(GO) mod tidy
.PHONY: mod mod-tidy

tools: 
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint
.PHONY: tools

test:
	$(GO) test -v ./...
lint:
	$(GOENV) golangci-lint run
test-all: test lint
.PHONY: test lint test-all

build:
	$(GO) build -o ./bin/$(NAME)
.PHONY: build

clean:
	$(GO) clean
	rm -rf bin static dist
.PHONY: clean