BINARY_NAME := anchor$(shell go env GOEXE)
BIN_DIR := bin

.PHONY: build install fmt vet test clean

build:
	go build -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/anchor

install: build
	$(BIN_DIR)/$(BINARY_NAME) install

fmt:
	gofmt -l -w .

vet:
	go vet ./...

test:
	go test ./...

clean:
	rm -rf $(BIN_DIR)
