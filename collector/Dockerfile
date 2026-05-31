FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN go build -o bin/collector ./cmd/collector
RUN go build -o bin/migrate ./cmd/migrate

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/bin/collector ./collector
COPY --from=builder /app/bin/migrate ./migrate

CMD ["./collector"]