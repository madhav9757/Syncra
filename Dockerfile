# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the server binary
RUN CGO_ENABLED=0 GOOS=linux go build -o syncra-server ./cmd/server/main.go

# Final stage
FROM alpine:3.18

WORKDIR /app

# Add certificates for secure DB connections
RUN apk add --no-cache ca-certificates

# Copy the binary from builder
COPY --from=builder /app/syncra-server .

# Set environment variables for production
ENV HEADLESS=true
ENV PORT=8080

EXPOSE 8080

# Command to run the server
CMD ["./syncra-server"]
