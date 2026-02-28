# GoVision

### *A Scalable, Event-Driven Object Detection Pipeline using Go, RabbitMQ and Roboflow*

---

## Overview

GoVision is an event-driven object detection platform designed to process images asynchronously and at scale. The system leverages machine learning models through the Roboflow API to analyze objects in images with high throughput and reliability.

This repository contains:
- **Upload API** — receives image uploads, stores them and enqueues processing jobs
- **Processing Worker** — consumes the queue, sends images to Roboflow for inference and returns detection results

---

## Purpose

The Upload API serves as the entry point for the GoVision system. Its key responsibilities include:

- **Image Reception**: Accepts image uploads via HTTP multipart/form-data
- **Validation**: Verifies file size, type, and integrity
- **Storage**: Persists images to external storage service (ImgBB)
- **Job Enqueueing**: Publishes messages to RabbitMQ for asynchronous processing
- **Resilience**: Ensures successful uploads are always queued for processing

The decoupled architecture enables multiple workers to consume the queue and process images in parallel, ensuring horizontal scalability.

---

## Technical Architecture

### Technology Stack

- **Language**: Go 1.23+
- **HTTP Framework**: Echo v4
- **Message Broker**: RabbitMQ (amqp091-go)
- **Database**: PostgreSQL (GORM)
- **Authentication**: JWT (golang-jwt/v5) + Refresh Token Rotation
- **Storage**: ImgBB API (HTTP-based)
- **ML Inference**: Roboflow Workflows API
- **ID Management**: ULID (Universally Unique Lexicographically Sortable Identifier)

### Core Components

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │ POST /v1/image/upload
       ▼
┌─────────────────────┐
│   Echo HTTP API     │
│  (Port 8080)        │
└──────┬──────────────┘
       │
       ├──► Validation (size, type)
       │
       ├──► Upload to ImgBB
       │
       └──► Publish to RabbitMQ
              │
              ▼
       ┌─────────────────┐
       │  RabbitMQ Queue  │
       │   (image_jobs)   │
       └──────┬──────────┘
              │
              ▼
       ┌─────────────────┐
       │  Worker          │
       │  (Consumer)      │
       └──────┬──────────┘
              │
              ├──► Create pending job (PostgreSQL)
              │
              └──► Send image URL to Roboflow Workflows API
                     │
                     ▼
              ┌──────────────────┐
              │  Detection       │
              │  Results (JSON)  │
              └──────┬───────────┘
                     │
                     ▼
              ┌──────────────────┐
              │  PostgreSQL      │
              │  (jobs +         │
              │   predictions)   │
              └──────────────────┘
```

### Processing Flow

<img width="641" height="355" alt="image" src="https://github.com/user-attachments/assets/900fd6e6-110c-45e1-903f-4654cad972d2" />

1. **Request**: Client sends image via POST multipart/form-data
2. **Validation**:
   - Max size: 5MB
   - Allowed types: JPEG, PNG, GIF
   - Content-type detection via magic bytes
3. **Storage**: Upload to ImgBB and retrieve public URL
4. **Job Creation**: Generate Job ID (ULID) for traceability
5. **Queue**: Publish job to RabbitMQ with metadata
6. **Response**: Return Job ID and "queued" status to client

### Security Middlewares

- **CORS**: Cross-origin access control
- **Body Limit**: 5MB per request maximum
- **Security Headers**: DNS prefetch control, COOP, COEP, Permissions Policy
- **Recovery**: Automatic panic recovery
- **Logging**: Structured logs with timestamps and request duration

---

## API Endpoints

### Authentication

#### `POST /v1/auth/register`

Register a new user.

**Request:**
```bash
curl -X POST http://localhost:8080/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "SecurePass1"}'
```

**Response (201 Created):**
```json
{"message": "User registered successfully"}
```

#### `POST /v1/auth/login`

Authenticate and receive tokens.

**Request:**
```bash
curl -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "SecurePass1"}'
```

**Response (200 OK):**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "a1b2c3d4e5f6...",
  "expires_in": 900
}
```

#### `POST /v1/auth/refresh`

Refresh an expired access token.

**Request:**
```bash
curl -X POST http://localhost:8080/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "a1b2c3d4e5f6..."}'
```

**Response (200 OK):**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "f6e5d4c3b2a1...",
  "expires_in": 900
}
```

### Protected Routes

All routes below require `Authorization: Bearer <access_token>` header.

#### `POST /v1/image/upload`

Upload an image for processing.

**Request:**
```bash
curl -X POST http://localhost:8080/v1/image/upload \
  -H "Authorization: Bearer <access_token>" \
  -F "file=@image.jpg"
```

**Response (202 Accepted):**
```json
{
  "job_id": "01JCXA1B2C3D4E5F6G7H8J9K0M",
  "status": "queued"
}
```

#### `GET /v1/jobs/:id`

Query job status and detection results.

**Request:**
```bash
curl http://localhost:8080/v1/jobs/01JCXA1B2C3D4E5F6G7H8J9K0M \
  -H "Authorization: Bearer <access_token>"
```

**Response (200 OK):**
```json
{
  "job_id": "01JCXA1B2C3D4E5F6G7H8J9K0M",
  "image_url": "https://i.ibb.co/abc123/image.jpg",
  "status": "completed",
  "processed_at": "2026-02-28T20:10:55Z",
  "created_at": "2026-02-28T20:10:50Z",
  "predictions": [
    {
      "x": 212.02,
      "y": 49.53,
      "width": 65.06,
      "height": 70.18,
      "confidence": 0.77,
      "class": "apple",
      "class_id": 1
    }
  ]
}
```

---

## Configuration

### Environment Variables

Create a `.env` file in the project root:

```env
# Server
API_PORT=8080

# Authentication
JWT_SECRET=your_jwt_secret_here

# Storage (ImgBB)
STORAGE_API_KEY=your_imgbb_api_key_here

# Message Queue (RabbitMQ)
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
RABBITMQ_QUEUE=image_jobs

# Database (PostgreSQL)
DATABASE_URL=postgresql://user:password@localhost:5432/govision?sslmode=disable

# Roboflow (Object Detection)
ROBOFLOW_API_KEY=your_roboflow_api_key_here
ROBOFLOW_WORKSPACE_ID=your_workspace_id
ROBOFLOW_WORKFLOW_ID=your_workflow_id
```

### Dependencies

```bash
go mod download
```

---

## Running

### Local Development

**API:**
```bash
go run ./api/cmd/server.go
```

The API will be available at `http://localhost:8080`

**Worker:**
```bash
go run ./worker/cmd/main.go
```

### Production Build

```bash
# API
go build -o bin/govision-api ./api/cmd/server.go
./bin/govision-api

# Worker
go build -o bin/govision-worker ./worker/cmd/main.go
./bin/govision-worker
```

### Docker (Future)

```bash
docker-compose up -d
```

---

## Project Structure

```
govision/
├── .env                          # Shared environment variables
├── go.mod                        # Unified Go module
├── go.sum
├── LICENSE
├── README.md
├── migrations/
│   ├── 001_create_tables.sql     # Jobs & predictions tables
│   └── 002_create_auth_tables.sql# Users & refresh tokens tables
├── api/
│   ├── cmd/
│   │   └── server.go             # API entry point
│   ├── internal/
│   │   ├── middlewares/
│   │   │   ├── security.go       # HTTP security middlewares
│   │   │   └── jwt.go            # JWT authentication middleware
│   │   ├── modules/
│   │   │   ├── auth/
│   │   │   │   ├── handler.go    # Auth HTTP handlers
│   │   │   │   ├── repository.go # User & token persistence
│   │   │   │   ├── service.go    # Auth business logic (JWT + bcrypt)
│   │   │   │   ├── types.go      # Auth models & DTOs
│   │   │   │   └── validator.go  # Input validations
│   │   │   ├── file/
│   │   │   │   ├── handler.go    # Upload HTTP handler
│   │   │   │   ├── service.go    # Upload business logic
│   │   │   │   ├── types.go      # DTOs
│   │   │   │   └── validator.go  # File validations
│   │   │   └── job/
│   │   │       ├── handler.go    # Job status HTTP handler
│   │   │       ├── repository.go # Job query persistence
│   │   │       ├── service.go    # Job query business logic
│   │   │       └── types.go      # Job models & DTOs
│   │   └── routes/
│   │       └── routes.go         # Route definitions
│   ├── pkg/
│   │   └── utils/
│   │       └── sendRequest.go    # HTTP request utility
│   └── services/
│       ├── postgres/
│       │   ├── migrations.go     # Auto-migration runner
│       │   └── postgres.go       # PostgreSQL connection (GORM)
│       ├── rabbitmq/
│       │   ├── connection.go     # RabbitMQ connection
│       │   ├── interface.go      # Publisher interface
│       │   └── publish.go        # Job publisher
│       └── storage/
│           └── getImageUrl.go    # ImgBB storage client
└── worker/
    ├── cmd/
    │   └── main.go               # Worker entry point
    └── internal/
        ├── domain/
        │   ├── job.go            # Job message type
        │   ├── models.go         # Database models (GORM)
        │   └── prediction.go     # Roboflow response types
        ├── repository/
        │   ├── repository.go     # Repository interface
        │   └── postgres/
        │       └── prediction_repository.go # PostgreSQL implementation
        ├── services/
        │   ├── postgres/
        │   │   └── postgres.go   # PostgreSQL connection (GORM)
        │   ├── rabbitmq/
        │   │   ├── connection.go  # RabbitMQ connection
        │   │   └── consumer.go   # Queue consumer
        │   └── roboflow/
        │       └── roboflow.go   # Roboflow Workflows API client
        └── worker/
            └── worker.go         # Job processing logic
```

---

## RabbitMQ Message Format

When an upload succeeds, the API publishes a message in the format:

```json
{
  "job_id": "01JCXA1B2C3D4E5F6G7H8J9K0M",
  "image_url": "https://i.ibb.co/abc123/image.jpg"
}
```

**Metadata:**
- `MessageId`: Job ID (ULID)
- `Timestamp`: Job creation timestamp
- `ContentType`: application/json

---

## Pipeline Roadmap

1. ✅ **Upload API** — Image reception, validation, storage and job enqueueing
2. ✅ **Processing Worker** — RabbitMQ consumer with Roboflow Workflows API integration
3. ✅ **Results Storage** — PostgreSQL persistence for jobs and predictions (GORM)
4. ✅ **Results API** — Query job status and detection results (`GET /v1/jobs/:id`)
5. ✅ **Database Migrations** — Auto-executed SQL migrations on API startup
6. ✅ **JWT Authentication** — Access tokens (15min) + refresh token rotation (7 days)
7. ⏳ **Dashboard** — Web interface for results visualization
8. ⏳ **Docker** — Containerized deployment with docker-compose

The event-driven architecture allows each service to be developed, tested, and scaled independently.

---

## Contributing

This project is under active development. Contributions are welcome!

---
