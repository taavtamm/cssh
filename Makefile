VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)

.PHONY: build install run test lint clean

build:
	go build -ldflags "$(LDFLAGS)" -o cssh .

install:
	go install -ldflags "$(LDFLAGS)" .
	@echo "Installed to $$(go env GOPATH)/bin/cssh"
	@echo "Add \$$GOPATH/bin to PATH if not already there"
	@echo "Or run: sudo cp \$$(go env GOPATH)/bin/cssh /usr/local/bin/"

run:
	go run .

test:
	go test ./...

lint:
	gofmt -l . && test -z "$$(gofmt -l .)"
	go vet ./...

clean:
	rm -f cssh
