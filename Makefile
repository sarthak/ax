BIN := ax

.PHONY: build run clean test

build:
	go build -o $(BIN) .

run:
	go run .

test:
	go test ./...

clean:
	rm -f $(BIN)
