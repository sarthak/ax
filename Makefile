BIN := ax

.PHONY: build run clean test test-integration lint fmt

build:
	go build -o $(BIN) .

run:
	go run .

test:
	go test ./...

test-integration: build
	go test -tags integration -v -count=1 -timeout 60s ./e2e/

clean:
	rm -f $(BIN)

lint:
	golangci-lint run ./...

fmt:
	goimports -w .
	go mod tidy
