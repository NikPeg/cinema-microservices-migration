# Proxy Service (API Gateway)

## Overview

The Proxy Service implements the Strangler Fig pattern to enable gradual migration from the monolithic architecture to microservices. It acts as an API Gateway, routing requests to either the monolith or the new microservices based on configuration.

## Features

- **Strangler Fig Pattern**: Gradual migration from monolith to microservices
- **Feature Flags**: Control traffic routing using environment variables
- **Percentage-based Routing**: Route a percentage of traffic to new services
- **Logging**: Comprehensive request/response logging
- **CORS Support**: Built-in CORS middleware
- **Health Checks**: Health endpoint for monitoring

## Architecture

```
Client Request → Proxy Service → Decision Logic → Route to:
                                                   ├── Monolith (legacy)
                                                   ├── Movies Service (new)
                                                   └── Events Service (new)
```

## Configuration

The service is configured through environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Port to listen on | `8000` |
| `MONOLITH_URL` | URL of the monolith service | `http://localhost:8080` |
| `MOVIES_SERVICE_URL` | URL of the movies microservice | `http://localhost:8081` |
| `EVENTS_SERVICE_URL` | URL of the events microservice | `http://localhost:8082` |
| `GRADUAL_MIGRATION` | Enable gradual migration | `true` |
| `MOVIES_MIGRATION_PERCENT` | Percentage of traffic to route to movies service | `50` |

## Routing Rules

### Movies Endpoints (`/api/movies*`)
- If `GRADUAL_MIGRATION` is `true`:
  - Routes `MOVIES_MIGRATION_PERCENT`% of traffic to the movies microservice
  - Routes remaining traffic to the monolith
- If `GRADUAL_MIGRATION` is `false`:
  - Routes all traffic to the movies microservice

### Events Endpoints (`/api/events*`)
- Always routes to the events microservice

### All Other Endpoints
- Routes to the monolith

## Running Locally

### Prerequisites
- Go 1.21 or higher
- Running instances of:
  - Monolith service (port 8080)
  - Movies service (port 8081)
  - Events service (port 8082)

### Build and Run

```bash
# Build the service
go build -o proxy-service .

# Run with default configuration
./proxy-service

# Run with custom configuration
MOVIES_MIGRATION_PERCENT=75 ./proxy-service
```

### Using Docker

```bash
# Build the Docker image
docker build -t proxy-service .

# Run the container
docker run -p 8000:8000 \
  -e MONOLITH_URL=http://monolith:8080 \
  -e MOVIES_SERVICE_URL=http://movies-service:8081 \
  -e EVENTS_SERVICE_URL=http://events-service:8082 \
  -e GRADUAL_MIGRATION=true \
  -e MOVIES_MIGRATION_PERCENT=50 \
  proxy-service
```

### Using Docker Compose

The service is already configured in the main `docker-compose.yml`:

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f proxy-service

# Test the proxy
curl http://localhost:8000/api/movies
```

## Testing Migration

### Test Gradual Migration

1. Set migration to 0% (all traffic to monolith):
   ```bash
   docker-compose down
   MOVIES_MIGRATION_PERCENT=0 docker-compose up -d
   curl http://localhost:8000/api/movies
   # Check logs to verify routing to monolith
   ```

2. Set migration to 50% (split traffic):
   ```bash
   docker-compose down
   MOVIES_MIGRATION_PERCENT=50 docker-compose up -d
   # Make multiple requests
   for i in {1..10}; do curl http://localhost:8000/api/movies; done
   # Check logs to see traffic distribution
   ```

3. Set migration to 100% (all traffic to microservice):
   ```bash
   docker-compose down
   MOVIES_MIGRATION_PERCENT=100 docker-compose up -d
   curl http://localhost:8000/api/movies
   # Check logs to verify routing to movies service
   ```

## API Endpoints

### Health Check
```
GET /health
```
Returns: `Strangler Fig Proxy is healthy`

### Proxied Endpoints
All endpoints from the monolith and microservices are available through the proxy:

- `/api/users` - User management (monolith)
- `/api/movies` - Movies catalog (monolith or microservice based on configuration)
- `/api/payments` - Payment processing (monolith)
- `/api/subscriptions` - Subscription management (monolith)
- `/api/events/*` - Event processing (events microservice)

## Monitoring

The proxy logs all requests with the following information:
- HTTP method and path
- Source service (monolith or microservice)
- Migration percentage (for gradual migration endpoints)
- Response status code
- Request duration

Example log output:
```
2024-01-15 10:30:45 Incoming request: GET /api/movies
2024-01-15 10:30:45 Routing to movies microservice: /api/movies (migration: 75%)
2024-01-15 10:30:45 [GET] /api/movies 192.168.1.100:54321 - Status: 200 - Duration: 45ms
```

## Development

### Project Structure
```
proxy/
├── main.go           # Main application code
├── go.mod           # Go module definition
├── go.sum           # Go module checksums
├── Dockerfile       # Docker build configuration
└── README.md        # This file
```

### Adding New Routes

To add routing for a new microservice:

1. Add the service URL to configuration
2. Update the `ServeHTTP` method to handle the new route
3. Create a new reverse proxy instance for the service

Example:
```go
// In config
PaymentServiceURL string

// In ServeHTTP
if strings.HasPrefix(path, "/api/payments") {
    ps.paymentProxy.ServeHTTP(w, r)
    return
}
```

## Troubleshooting

### Service Not Reachable
- Check that all backend services are running
- Verify URLs in environment variables
- Check Docker network connectivity

### Incorrect Routing
- Check `GRADUAL_MIGRATION` setting
- Verify `MOVIES_MIGRATION_PERCENT` value
- Review proxy logs for routing decisions

### Performance Issues
- Monitor backend service health
- Check for network latency
- Review timeout settings

## License

This service is part of the CinemaAbyss project.
