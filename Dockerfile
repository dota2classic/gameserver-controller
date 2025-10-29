# Use the official Golang image for building
FROM golang:1.25-alpine AS builder

# Set working directory inside container
WORKDIR /root

# Copy go.mod and go.sum if you have dependencies (optional)
# COPY go.mod go.sum ./
# RUN go mod download

# Copy the source code
COPY . .

# Build the Go binary
RUN go build -o controller cmd/controller/main.go

# -----------------------------

# Use a minimal image for running
FROM alpine:latest

# Set working directory
WORKDIR /root

# Copy the binary from builder
COPY --from=builder /root/controller .

# Expose port 7777
EXPOSE 7777

# Run the app
CMD ["./controller"]
