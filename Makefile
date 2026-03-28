BIN := ax

.PHONY: build run clean test lint fmt

build:
	go build -o $(BIN) .

run:
	go run .

test:
	go test ./...

clean:
	rm -f $(BIN)

lint:
	golangci-lint run ./...

fmt:
	goimports -w .
	go mod tidy
