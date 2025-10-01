# Events Service

## Overview

The Events Service is a microservice that handles event-driven communication in the CinemaAbyss system using Apache Kafka. It provides both producer and consumer functionality for movie, user, and payment events.

## Features

- **Event Publishing**: Publishes events to Kafka topics
- **Event Consumption**: Consumes and processes events from Kafka topics
- **Multiple Event Types**: Supports Movie, User, and Payment events
- **RESTful API**: HTTP endpoints for creating events
- **Automatic Processing**: Built-in consumers that process events in real-time
- **Logging**: Comprehensive event logging for debugging and monitoring

## Architecture

```
HTTP Request → Events Service → Kafka Producer → Kafka Topic
                                                        ↓
                                              Kafka Consumer → Event Processing → Logs
```

## Event Types

### Movie Events
- Actions: viewed, rated, added, updated, deleted
- Topic: `movie-events`

### User Events
- Actions: registered, logged_in, updated_profile, deleted
- Topic: `user-events`

### Payment Events
- Status: completed, failed, pending, refunded
- Topic: `payment-events`

## Configuration

Environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP server port | `8082` |
| `KAFKA_BROKERS` | Kafka broker addresses | `localhost:9092` |

## API Endpoints

### Health Check
```
GET /api/events/health
```
Returns service health status.

### Create Movie Event
```
POST /api/events/movie
Content-Type: application/json

{
  "movie_id": 1,
  "title": "The Matrix",
  "action": "viewed",
  "user_id": 123,
  "rating": 9.5,
  "genres": ["Sci-Fi", "Action"],
  "description": "A computer hacker learns about the true nature of reality"
}
```

### Create User Event
```
POST /api/events/user
Content-Type: application/json

{
  "user_id": 123,
  "username": "john_doe",
  "email": "john@example.com",
  "action": "registered",
  "timestamp": "2025-01-01T00:00:00Z"
}
```

### Create Payment Event
```
POST /api/events/payment
Content-Type: application/json

{
  "payment_id": 456,
  "user_id": 123,
  "amount": 99.99,
  "status": "completed",
  "timestamp": "2025-01-01T00:00:00Z",
  "method_type": "credit_card"
}
```

## Running Locally

### Prerequisites
- Go 1.21 or higher
- Apache Kafka running on localhost:9092

### Build and Run

```bash
# Install dependencies
go mod download

# Build the service
go build -o events-service .

# Run the service
./events-service

# Or run directly
go run main.go
```

### Using Docker

```bash
# Build the Docker image
docker build -t events-service .

# Run the container
docker run -p 8082:8082 \
  -e KAFKA_BROKERS=kafka:9092 \
  events-service
```

### Using Docker Compose

The service is configured in the main `docker-compose.yml`:

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f events-service

# Test the service
curl -X POST http://localhost:8082/api/events/movie \
  -H "Content-Type: application/json" \
  -d '{"movie_id": 1, "title": "Test Movie", "action": "viewed"}'
```

## Development

### Project Structure
```
events/
├── main.go          # Main application code
├── go.mod          # Go module definition
├── go.sum          # Go module checksums
├── Dockerfile      # Docker build configuration
└── README.md       # This file
```

### Dependencies
- `github.com/segmentio/kafka-go` - Kafka client library
- `github.com/google/uuid` - UUID generation

### Testing

```bash
# Run unit tests
go test ./...

# Run with coverage
go test -cover ./...

# Run integration tests (requires Kafka)
KAFKA_BROKERS=localhost:9092 go test -tags=integration ./...
```

## Monitoring

The service logs all events with the following prefixes:
- `[PRODUCER]` - Event publishing logs
- `[CONSUMER]` - Event consumption logs
- `[EVENT PROCESSED]` - Processed event details

Example log output:
```
2025/01/01 10:00:00 [PRODUCER] Published movie event: abc-123, Action: viewed, MovieID: 1
2025/01/01 10:00:00 [CONSUMER] Topic: movie-events, Partition: 0, Offset: 42, Key: abc-123
2025/01/01 10:00:00 [EVENT PROCESSED] Type: movie, ID: abc-123, Timestamp: 2025-01-01T10:00:00Z
```

## Kafka Topics

The service automatically creates the following topics if they don't exist:
- `movie-events` - Movie-related events
- `user-events` - User-related events
- `payment-events` - Payment-related events

Each topic is configured with:
- Partitions: 1 (can be increased for scalability)
- Replication Factor: 1 (should be increased in production)

## Error Handling

- Connection errors are logged and retried
- Invalid JSON payloads return 400 Bad Request
- Kafka publishing failures return 500 Internal Server Error
- Consumer errors are logged but don't stop the consumer

## Performance Considerations

- Events are processed asynchronously
- Multiple consumers run in parallel for different topics
- Graceful shutdown ensures all events are processed
- Connection pooling for Kafka producers

## Security Considerations

- No authentication implemented (add JWT/OAuth in production)
- No encryption for Kafka messages (add TLS in production)
- Input validation for all event payloads
- Rate limiting should be added for production use

## License

This service is part of the CinemaAbyss project.
