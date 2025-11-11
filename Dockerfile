FROM golang:1.25.4-bookworm AS builder

WORKDIR /app

RUN apt-get update && apt-get install -y build-essential

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=1

RUN go build -v -o /ingestion-service ./cmd

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates libstdc++6 && \
    rm -rf /var/lib/apt/lists/*


COPY --from=builder /ingestion-service /usr/local/bin/ingestion-service

RUN mkdir -p /data
ENV DUCKDB_PATH="/data/security.db"

CMD ["/usr/local/bin/ingestion-service"]