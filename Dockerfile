FROM golang:alpine AS builder

WORKDIR /build

ADD go.mod go.sum ./

RUN go mod download
RUN go mod verify

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/exporter ./cmd/mikrotik-prom-exporter/*.go

FROM alpine AS final

WORKDIR /app

COPY --from=builder /app/exporter /app
COPY --from=builder /build/resources /app/resources

EXPOSE 9100/tcp
CMD ["/app/exporter", "export"]
