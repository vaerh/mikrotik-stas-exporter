FROM golang:1.22 as builder

WORKDIR /usr/src/app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN make build


FROM  golang:1.22-bookworm as final

WORKDIR /app

COPY --from=builder /usr/src/app/bin/mikrotik-prom-exporter /app

CMD ["/app/mikrotik-prom-exporter", "export"]


