BINARY := sf2subset

.PHONY: build test check install clean

build:
	go build -o $(BINARY) .

test:
	go test ./...

check:
	test -z "$$(gofmt -l .)" || { echo "gofmt needed:"; gofmt -l .; exit 1; }
	go vet ./...
	go test ./...

install:
	go install .

clean:
	rm -f $(BINARY)
