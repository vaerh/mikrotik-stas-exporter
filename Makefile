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

.PHONY:lint
lint:
	docker run -t --rm -v $$(pwd):/app -w /app golangci/golangci-lint:v1.58.1 golangci-lint run -v