FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod ./
# Use go mod tidy and download if you have dependencies
RUN go mod download

# Copy source code
COPY consumer.go .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o consumer .

# Use a small alpine image for the final container
FROM alpine:3.18

WORKDIR /app

# Install CA certificates for potential HTTPS connections
RUN apk --no-cache add ca-certificates

# Copy the binary from the builder stage
COPY --from=builder /app/consumer .

# Expose the application port
EXPOSE 8081

# Create a non-root user to run the application
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

# Command to run the application
CMD ["./consumer"]
