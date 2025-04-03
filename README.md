# AWS Service Connect Testing with Go Microservices

This project contains two simple Go microservices for testing AWS Service Connect functionality both locally and in ECS.

## Components

1. **Producer Service**: Creates and stores messages
2. **Consumer Service**: Communicates with the producer to fetch and create messages

## Project Structure

```
.
├── producer.go           # Producer service code
├── consumer.go           # Consumer service code
├── producer.Dockerfile   # Dockerfile for the producer service
├── consumer.Dockerfile   # Dockerfile for the consumer service
├── go.mod                # Go module file
└── docker-compose.yml    # Docker Compose configuration for local testing
```

## Local Testing

### Prerequisites

- Docker
- Docker Compose
- Go 1.21+ (for development)

### Running Locally

1. Clone this repository
2. Create the files with the provided code
3. Start the services:

```bash
docker-compose up --build
```

This will start both services:
- Producer: http://localhost:8080
- Consumer: http://localhost:8081

### Testing the Services

#### Producer Service Endpoints:

- `GET /`: Get service status
- `GET /health`: Health check endpoint
- `GET /messages`: Get all messages
- `POST /messages`: Create a new message

Example:
```bash
# Create a message
curl -X POST http://localhost:8080/messages \
  -H "Content-Type: application/json" \
  -d '{"content":"Hello, Service Connect!"}'

# Get all messages
curl http://localhost:8080/messages
```

#### Consumer Service Endpoints:

- `GET /`: Get service status
- `GET /health`: Health check endpoint
- `GET /fetch-messages`: Fetch all messages from the producer
- `POST /create-message`: Create a message via the producer

Example:
```bash
# Create a message via the consumer
curl -X POST http://localhost:8081/create-message \
  -H "Content-Type: application/json" \
  -d '{"content":"Hello from the consumer!"}'

# Fetch all messages
curl http://localhost:8081/fetch-messages
```

## Deploying to AWS ECS with Service Connect

To deploy these services to AWS ECS with Service Connect:

1. Push the Docker images to Amazon ECR
2. Create an ECS cluster with Service Connect enabled
3. Define an ECS task definition for each service with Service Connect configuration
4. Create ECS services using these task definitions and enable Service Connect

## Service Connect Configuration in ECS

The key Service Connect configuration in your ECS task definitions would look like:

### Producer Service:

```json
"serviceConnectConfiguration": {
  "enabled": true,
  "namespace": "your-service-connect-namespace",
  "services": [
    {
      "portName": "producer-port",
      "discoveryName": "producer",
      "clientAliases": [
        {
          "port": 8080
        }
      ]
    }
  ]
}
```

### Consumer Service:

```json
"serviceConnectConfiguration": {
  "enabled": true,
  "namespace": "your-service-connect-namespace"
}
```

In the ECS environment:
- The consumer service will use `http://producer:8080` to connect to the producer service
- AWS Service Connect will handle service discovery and routing

This setup demonstrates how Service Connect simplifies microservice communication by providing DNS-based service discovery.
