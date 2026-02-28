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
- **Storage**: ImgBB API (HTTP-based)
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
              ├──► Download image from URL
              │
              └──► Send to Roboflow API
                     │
                     ▼
              ┌──────────────────┐
              │  Detection       │
              │  Results (JSON)  │
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

### `POST /v1/image/upload`

Upload an image for processing.

**Request:**
```bash
curl -X POST http://localhost:8080/v1/image/upload \
  -F "file=@image.jpg"
```

**Response (202 Accepted):**
```json
{
  "job_id": "01JCXA1B2C3D4E5F6G7H8J9K0M",
  "status": "queued"
}
```

**Possible Errors:**
- `400 Bad Request`: Invalid file, too large, or unsupported type
- `500 Internal Server Error`: Storage or processing error
- `502 Bad Gateway`: Failed to communicate with storage service

---

## Configuration

### Environment Variables

Create a `.env` file in the project root:

```env
# Server
API_PORT=8080

# Storage (ImgBB)
STORAGE_API_KEY=your_imgbb_api_key_here

# Message Queue (RabbitMQ)
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
RABBITMQ_QUEUE=image_jobs

# Roboflow (Object Detection)
ROBOFLOW_API_KEY=your_roboflow_api_key_here
ROBOFLOW_MODEL=your-project/1
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
├── api/
│   ├── cmd/
│   │   └── server.go             # API entry point
│   ├── internal/
│   │   ├── middlewares/
│   │   │   └── security.go       # HTTP security middlewares
│   │   ├── modules/
│   │   │   └── file/
│   │   │       ├── handler.go    # HTTP handler
│   │   │       ├── service.go    # Upload business logic
│   │   │       ├── types.go      # DTOs
│   │   │       └── validator.go  # File validations
│   │   └── routes/
│   │       └── routes.go         # Route definitions
│   ├── pkg/
│   │   └── utils/
│   │       └── sendRequest.go    # HTTP request utility
│   └── services/
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
        │   └── prediction.go     # Roboflow response types
        ├── services/
        │   ├── rabbitmq/
        │   │   ├── connection.go # RabbitMQ connection
        │   │   └── consumer.go   # Queue consumer
        │   └── roboflow/
        │       └── roboflow.go   # Roboflow API client
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
2. ✅ **Processing Worker** — RabbitMQ consumer with Roboflow integration
3. ⏳ **Results API** — Query job status and detection results
4. ⏳ **Results Storage** — Persist detections (PostgreSQL/MongoDB)
5. ⏳ **Dashboard** — Web interface for results visualization

The event-driven architecture allows each service to be developed, tested, and scaled independently.

---

## Contributing

This project is under active development. Contributions are welcome!

---
