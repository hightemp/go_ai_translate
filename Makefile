BINARY_NAME=go_ai_translate
GO=go
GOFMT=gofmt
GOTEST=$(GO) test
GOBUILD=$(GO) build

.PHONY: all
all: fmt test build

.PHONY: fmt
fmt:
	$(GOFMT) -w .

.PHONY: test
test:
	$(GOTEST) -v ./...

.PHONY: build
build:
	$(GOBUILD) -o $(BINARY_NAME) .

.PHONY: build-static
build-static:
	CGO_ENABLED=0 $(GOBUILD) -ldflags="-extldflags=-static" -o $(BINARY_NAME) .

.PHONY: clean
clean:
	rm -f $(BINARY_NAME)