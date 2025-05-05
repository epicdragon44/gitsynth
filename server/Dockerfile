# Use the official Go image as base
FROM golang:1.24-bookworm as builder

WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server .

# Use a small image for the final container
FROM fedora:latest

# Install ca-certificates for HTTPS
RUN dnf -y update && dnf -y install ca-certificates && dnf clean all

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/server /app/server

# Copy templates directory
COPY --from=builder /app/templates /app/templates

# Expose the port the server runs on
EXPOSE 8080

# Command to run the executable
CMD ["/app/server"]