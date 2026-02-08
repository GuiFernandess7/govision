# GoVision

### *A Scalable, Event-Driven Object Detection Pipeline using Go, RabbitMQ and Roboflow*

---

## Overview

GoVision is an event-driven object detection platform designed to process images asynchronously and at scale. The system leverages machine learning models through the Roboflow API to analyze objects in images with high throughput and reliability.

This repository contains the **Image Upload API**, the first service in the GoVision pipeline, responsible for receiving image uploads, storing them, and enqueueing processing jobs for downstream services.

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
       │  RabbitMQ Queue │
       │   (image_jobs)  │
       └─────────────────┘
              │
              ▼
       [Next pipeline services]
```

### Processing Flow

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

Create a `.env` file in the `api/` directory:

```env
# Server
API_PORT=8080

# Storage (ImgBB)
STORAGE_API_KEY=your_imgbb_api_key_here

# Message Queue (RabbitMQ)
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
RABBITMQ_QUEUE=image_jobs
```

### Dependencies

```bash
cd api
go mod download
```

---

## Running the API

### Local Development

```bash
cd api
go run cmd/app/server.go
```

The API will be available at `http://localhost:8080`

### Production Build

```bash
cd api
go build -o bin/govision-api cmd/app/server.go
./bin/govision-api
```

### Docker (Future)

```bash
docker-compose up -d
```

---

## Project Structure

```
api/
├── cmd/app/
│   └── server.go              # Application entry point
├── internal/
│   ├── modules/file/          # File upload module
│   │   ├── handler.go         # HTTP handler
│   │   ├── types.go           # DTOs
│   │   └── validator.go       # Validations
│   ├── middlewares/           # HTTP middlewares
│   │   └── security.go
│   └── routes/                # Route definitions
│       └── routes.go
├── services/
│   ├── rabbitmq/              # RabbitMQ client
│   │   ├── connection.go
│   │   ├── interface.go
│   │   └── publish.go
│   └── storage/               # Storage service
│       └── getImageUrl.go
├── pkg/utils/                 # Reusable utilities
│   └── sendRequest.go
├── go.mod
└── go.sum
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

This is the **first service** in the GoVision pipeline. Upcoming components include:

1. ✅ **Upload API** (This service)
2. ⏳ **Processing Worker**: RabbitMQ consumer that processes images
3. ⏳ **Detection Service**: Roboflow integration for object detection
4. ⏳ **Results API**: Query job status and results
5. ⏳ **Results Storage**: Persist detections (PostgreSQL/MongoDB)
6. ⏳ **Dashboard**: Web interface for results visualization

The event-driven architecture allows each service to be developed, tested, and scaled independently.

---

## Contributing

This project is under active development. Contributions are welcome!

---
