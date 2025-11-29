# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o poc ./services/poc/main.go

# Runtime stage
FROM scratch

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/poc .

# Expose port
EXPOSE 8080

# Run the application
CMD ["./poc"]
