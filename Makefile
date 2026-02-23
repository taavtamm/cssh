.PHONY: build install run clean

build:
	go build -o cssh ./...

install:
	go install ./...
	@echo "Installed to $$(go env GOPATH)/bin/cssh"
	@echo "Add \$$GOPATH/bin to PATH if not already there"
	@echo "Or run: sudo cp \$$(go env GOPATH)/bin/cssh /usr/local/bin/"

run:
	go run .

clean:
	rm -f cssh
