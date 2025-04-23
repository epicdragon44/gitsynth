# Use a minimal Alpine Linux image
FROM alpine:3.19

# Set working directory
WORKDIR /app

# Install necessary runtime dependencies
RUN apk add --no-cache ca-certificates

# Copy the pre-built binary from the host
COPY bin/gitsynth-server /app/

# Make the binary executable
RUN chmod +x /app/gitsynth-server

# Expose the port the server listens on
EXPOSE 8080

# Set environment to production
ENV GO_ENV=production

# Run the binary
CMD ["/app/gitsynth-server"]