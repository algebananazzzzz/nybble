BIN := nybble

build:
	go build -trimpath -o $(BIN) ./cmd/nybble

test:
	go test ./...

run:
	go run ./cmd/nybble

fmt:
	gofmt -w .

vet:
	go vet ./...

# CI parity: fail if anything is unformatted, then vet + test.
lint:
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then echo "unformatted:"; echo "$$unformatted"; exit 1; fi
	go vet ./...
	go test ./...

tidy:
	go mod tidy

# Local dry-run of the full release build (needs goreleaser installed).
release-snapshot:
	goreleaser release --snapshot --clean

# Re-render the demo gif from the recorded cast (needs agg installed).
demo:
	agg --theme asciinema docs/demo.cast docs/demo.gif

clean:
	rm -f $(BIN)
	rm -rf dist

.PHONY: build test run fmt vet lint tidy release-snapshot demo clean
