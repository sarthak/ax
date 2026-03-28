BIN := ax

.PHONY: build run clean

build:
	go build -o $(BIN) .

run:
	go run .

clean:
	rm -f $(BIN)
