# ---- Build Stage ----
FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build for Linux
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o server main.go

# ---- Runtime Stage ----
FROM debian:bullseye-slim

WORKDIR /app

# Install nmap
RUN apt-get update && apt-get install -y nmap && apt-get clean

# Copy binary and docs
COPY --from=builder /app/server .
COPY --from=builder /app/docs ./docs

EXPOSE 8080

ENTRYPOINT ["./server"]
