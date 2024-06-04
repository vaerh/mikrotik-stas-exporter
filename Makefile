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

.PHONY: start-dev-router
start-dev-router:
	docker compose up -d

.PHONY: configure-router
configure-router:
	ROS_USERNAME="admin" \
	ROS_PASSWORD="" \
	ROS_IP_ADDRES="$$(docker inspect mikrotik-prom-exporter-routeros-1 | jq -r '.[0].NetworkSettings.Networks."mikrotik-prom-exporter_default".IPAddress ')" \
	go run .github/scripts/setup_routeros.go

.PHONY: start-dev-env
start-dev-env: start-dev-router configure-router

.PHONY: stop-dev-env
stop-dev-env:
	docker compose down


