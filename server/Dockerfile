# Use Docker-in-Docker image as base
FROM docker:dind

# Set working directory
WORKDIR /app

# Install necessary runtime dependencies
RUN apk add --no-cache ca-certificates \
    go \
    git \
    musl-dev \
    bash

# Copy the pre-built binary from the host
COPY bin/gitsynth-server /app/

# Make the binary executable
RUN chmod +x /app/gitsynth-server

# Expose the port the server listens on
EXPOSE 8080

# Set environment to production
ENV GO_ENV=production

# Create entrypoint script to start Docker daemon and then the server
RUN echo '#!/bin/sh\n\
echo "Starting Docker daemon..."\n\
dockerd &\n\
echo "Waiting for Docker daemon to start..."\n\
sleep 5\n\
echo "Starting GitSynth server..."\n\
/app/gitsynth-server' > /app/entrypoint.sh && \
    chmod +x /app/entrypoint.sh

# Run the entrypoint script
CMD ["/app/entrypoint.sh"]