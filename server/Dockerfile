# Start from the official Fedora image
FROM fedora:latest

# Install required packages
RUN dnf -y update && \
    dnf -y install \
    make \
    golang \
    git \
    nodejs \
    npm \
    && dnf clean all

# Install gitsynth npm package globally
RUN npm install -g gitsynth

# Set up working directory
WORKDIR /app

# Copy the entire server directory
COPY . .

# Build the application
RUN make build

# Expose port (adjust as needed based on your config)
EXPOSE 8080

# Run the server
CMD ["./bin/gitsynth-server"]
