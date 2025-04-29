FROM node:20-slim

# Install dependencies for node-fetch
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /usr/src/app

# Create and use non-root user for security
RUN groupadd -r gitsynth && \
    useradd -r -g gitsynth -m -d /home/gitsynth gitsynth && \
    chown -R gitsynth:gitsynth /usr/src/app

# Create a directory for the app
RUN mkdir -p /usr/src/app && chown -R gitsynth:gitsynth /usr/src/app
USER gitsynth

# Copy package files and install dependencies
COPY --chown=gitsynth:gitsynth package*.json ./
RUN npm ci --production && \
    npm cache clean --force

# Set environment variables
ENV NODE_ENV="production"
ENV PORT=3000

# Copy application code
COPY --chown=gitsynth:gitsynth . .

# Transpile TypeScript code
RUN npm run build

# Expose the port the app runs on
EXPOSE 3000

# Run the bot
CMD [ "npm", "start" ]
