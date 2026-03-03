.PHONY: test coverage lint fmt doc build clean help

test:
	go test ./...

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

lint:
	golangci-lint run ./...

fmt:
	go fmt ./...

doc:
	go doc -all ./...

build:
	go build ./...

clean:
	rm -f coverage.out coverage.html

help:
	@echo "Available targets:"
	@echo "  test      - Run all tests"
	@echo "  coverage  - Generate coverage report (coverage.html)"
	@echo "  lint      - Run golangci-lint"
	@echo "  fmt       - Format all Go files"
	@echo "  doc       - Show all documentation"
	@echo "  build     - Build all packages"
	@echo "  clean     - Remove generated files"