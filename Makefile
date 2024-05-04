

.PHONY: test
test:
	go test --race ./...

.PHONY: fmt
fmt:
	go mod tidy
	go fmt ./...

.PHONY: build
build:
	go build -o bin ./...