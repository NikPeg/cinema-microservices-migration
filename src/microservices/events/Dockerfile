# Temporary placeholder for events service
FROM alpine:latest

# Install basic tools
RUN apk --no-cache add curl

# Create working directory
WORKDIR /app

# Add a simple health check script
RUN echo '#!/bin/sh' > /app/server.sh && \
    echo 'echo "Events service placeholder starting..."' >> /app/server.sh && \
    echo 'while true; do' >> /app/server.sh && \
    echo '  echo "Events service is running (placeholder)..."' >> /app/server.sh && \
    echo '  sleep 30' >> /app/server.sh && \
    echo 'done' >> /app/server.sh && \
    chmod +x /app/server.sh

# Expose port
EXPOSE 8082

# Run the placeholder script
CMD ["/app/server.sh"]
