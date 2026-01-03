# API Documentation 
Complete reference for GoQueue API endpoints, request/response formats, and status codes.
## Base URL

```
http://localhost:8080
```

## Authentication

Currently, no authentication is required.

## Common Response Formats

### Success Response
```json
{
  "id": 1,
  "queue": "email",
  "type": "send_email",
  "status": "queued"
}
```

### Error Response
```json
{
  "error": "error message",
  "fields": {
    "field_name": "validation error detail"
  }
}
```

---

## Health Check Endpoints

### Health Check

Check if the API is running.

**Endpoint:** `GET /health`

**Response:** `200 OK`
```json
{
  "status": "ok"
}
```

### Database Health Check

Check if the database connection is healthy.

**Endpoint:** `GET /health/db`

**Response:** `200 OK`
```json
{
  "status": "ok"
}
```

**Error Response:** `503 Service Unavailable`
```json
{
  "error": "database is unavailable"
}
```

---

## Job Endpoints

### Create Job

Create a new job in the queue.

**Endpoint:** `POST /jobs/create`

**Headers:**
```
Content-Type: application/json
```

**Request Body:**
```json
{
  "queue": "email",
  "type": "send_email",
  "payload": {
    "to": "user@example.com",
    "subject": "Welcome",
    "body": "Hello World"
  },
  "max_retries": 3
}
```

**Parameters:**
- `queue` (string, required): Queue name. Allowed: `default`, `email`, `webhooks`
- `type` (string, required): Job type. Allowed: `send_email`, `process_payment`, `send_webhook`
- `payload` (object, required): Job-specific payload (see Job Types section)
- `max_retries` (integer, optional): Maximum retry attempts (0-20). Default: 3

**Response:** `201 Created`
```json
{
  "queue": "email",
  "type": "send_email",
  "payload": {
    "to": "user@example.com",
    "subject": "Welcome",
    "body": "Hello World"
  },
  "max_retries": 3
}
```

**Error Responses:**

`400 Bad Request` - Invalid request
```json
{
  "error": "invalid queue",
  "fields": {
    "provided": "invalid_queue",
    "allowed": ["default", "email", "webhooks"]
  }
}
```

`500 Internal Server Error` - Database error
```json
{
  "error": "failed to add job to database: connection failed"
}
```

---

### Get Job

Retrieve a job by its ID.

**Endpoint:** `GET /jobs/:id`

**Path Parameters:**
- `id` (integer, required): Job ID

**Response:** `200 OK`
```json
{
  "id": 1,
  "queue": "email",
  "type": "send_email",
  "payload": {
    "to": "user@example.com",
    "subject": "Welcome",
    "body": "Hello World"
  },
  "status": "queued",
  "attempts": 0,
  "max_retries": 3,
  "result": null,
  "error": "",
  "created_at": "2025-12-20T10:30:00Z",
  "updated_at": "2025-12-20T10:30:00Z"
}
```

**Error Responses:**

`400 Bad Request` - Invalid ID
```json
{
  "error": "Invalid ID"
}
```

`404 Not Found` - Job not found
```json
{
  "error": "Job not found"
}
```

---

### Update Job Status

Update the status of an existing job.

**Endpoint:** `PUT /jobs/:id/status`

**Path Parameters:**
- `id` (integer, required): Job ID

**Headers:**
```
Content-Type: application/json
```

**Request Body:**
```json
{
  "status": "running"
}
```

**Parameters:**
- `status` (string, required): New status. Common: `queued`, `running`, `completed`, `failed`

**Response:** `204 No Content`

**Error Responses:**

`400 Bad Request` - Invalid ID or missing status
```json
{
  "error": "invalid ID"
}
```

`500 Internal Server Error` - Update failed
```json
{
  "error": "failed to update job status"
}
```

---

### Increment Job Attempts

Increment the attempt counter for a job.

**Endpoint:** `POST /jobs/:id/increment`

**Path Parameters:**
- `id` (integer, required): Job ID

**Response:** `204 No Content`

**Error Responses:**

`400 Bad Request` - Invalid ID
```json
{
  "error": "invalid ID"
}
```

`500 Internal Server Error` - Increment failed
```json
{
  "error": "failed to increment job attempts"
}
```

---

### Save Job Result

Save the execution result and error message for a job.

**Endpoint:** `POST /jobs/:id/save`

**Path Parameters:**
- `id` (integer, required): Job ID

**Headers:**
```
Content-Type: application/json
```

**Request Body:**
```json
{
  "result": {
    "email_sent": true,
    "message_id": "msg_123"
  },
  "error": ""
}
```

**Parameters:**
- `result` (object, optional): Job execution result as JSON
- `error` (string, optional): Error message if job failed

**Response:** `204 No Content`

**Error Responses:**

`400 Bad Request` - Invalid ID or malformed JSON
```json
{
  "error": "invalid ID"
}
```

`500 Internal Server Error` - Save failed
```json
{
  "error": "failed to save job result"
}
```

---

### List Jobs

Retrieve all jobs for a specific queue.

**Endpoint:** `GET /jobs`

**Query Parameters:**
- `queue` (string, required): Queue name to filter by

**Example:**
```
GET /jobs?queue=email
```

**Response:** `200 OK`
```json
[
  {
    "id": 1,
    "queue": "email",
    "type": "send_email",
    "payload": {
      "to": "user1@example.com",
      "subject": "Welcome",
      "body": "Hello"
    },
    "status": "queued",
    "attempts": 0,
    "max_retries": 3,
    "created_at": "2025-12-20T10:30:00Z",
    "updated_at": "2025-12-20T10:30:00Z"
  }
]
```

**Error Responses:**

`400 Bad Request` - Missing queue parameter
```json
{
  "error": "queue parameter is required"
}
```

`500 Internal Server Error` - Query failed
```json
{
  "error": "failed to list jobs"
}
```

---

## Job Types and Payloads

### 1. Send Email

**Type:** `send_email`  
**Queue:** `email`

**Payload Schema:**
```json
{
  "to": "user@example.com",
  "subject": "Email Subject",
  "body": "Email body content"
}
```

**Validation Rules:**
- `to`: Required, valid email format
- `subject`: Required, non-empty string
- `body`: Required, non-empty string

**Example:**
```bash
curl -X POST http://localhost:8080/jobs/create \
  -H "Content-Type: application/json" \
  -d '{
    "queue": "email",
    "type": "send_email",
    "payload": {
      "to": "user@example.com",
      "subject": "Welcome to GoQueue",
      "body": "Thank you for signing up!"
    },
    "max_retries": 3
  }'
```

---

### 2. Process Payment

**Type:** `process_payment`  
**Queue:** `default`

**Payload Schema:**
```json
{
  "payment_id": "pay_123456",
  "user_id": "user_789",
  "amount": 99.99,
  "currency": "USD",
  "method": "card"
}
```

**Validation Rules:**
- `payment_id`: Required, string
- `user_id`: Required, string
- `amount`: Required, greater than 0
- `currency`: Required, exactly 3 characters (ISO 4217)
- `method`: Required, one of: `card`, `upi`, `netbanking`, `wallet`

**Example:**
```bash
curl -X POST http://localhost:8080/jobs/create \
  -H "Content-Type: application/json" \
  -d '{
    "queue": "default",
    "type": "process_payment",
    "payload": {
      "payment_id": "pay_123456",
      "user_id": "user_789",
      "amount": 99.99,
      "currency": "USD",
      "method": "card"
    },
    "max_retries": 5
  }'
```

---

### 3. Send Webhook

**Type:** `send_webhook`  
**Queue:** `webhooks`

**Payload Schema:**
```json
{
  "url": "https://example.com/webhook",
  "method": "POST",
  "headers": {
    "Authorization": "Bearer token",
    "Content-Type": "application/json"
  },
  "body": {
    "event": "user.created",
    "data": {}
  },
  "timeout": 10
}
```

**Validation Rules:**
- `url`: Required, valid URL format
- `method`: Required, one of: `POST`, `PUT`, `PATCH`
- `headers`: Optional, key-value pairs
- `body`: Required, JSON object
- `timeout`: Required, between 1-30 seconds

**Example:**
```bash
curl -X POST http://localhost:8080/jobs/create \
  -H "Content-Type: application/json" \
  -d '{
    "queue": "webhooks",
    "type": "send_webhook",
    "payload": {
      "url": "https://example.com/webhook",
      "method": "POST",
      "headers": {
        "Authorization": "Bearer secret_token"
      },
      "body": {
        "event": "user.created",
        "user_id": "123"
      },
      "timeout": 10
    },
    "max_retries": 3
  }'
```

---

## Status Codes

| Code | Description |
|------|-------------|
| 200 | Success |
| 201 | Created |
| 204 | No Content (successful update/delete) |
| 400 | Bad Request (validation error) |
| 404 | Not Found |
| 408 | Request Timeout |
| 500 | Internal Server Error |
| 503 | Service Unavailable (database down) |

---

## Rate Limiting

Currently not implemented. Will be added in future releases.

---

## Examples

### Complete Job Lifecycle

1. Create a job:
```bash
curl -X POST http://localhost:8080/jobs/create \
  -H "Content-Type: application/json" \
  -d '{
    "queue": "email",
    "type": "send_email",
    "payload": {
      "to": "user@example.com",
      "subject": "Test",
      "body": "Hello"
    }
  }'
```

2. Get job status:
```bash
curl http://localhost:8080/jobs/1
```

3. Update status to running:
```bash
curl -X PUT http://localhost:8080/jobs/1/status \
  -H "Content-Type: application/json" \
  -d '{"status": "running"}'
```

4. Increment attempts:
```bash
curl -X POST http://localhost:8080/jobs/1/increment
```

5. Save result:
```bash
curl -X POST http://localhost:8080/jobs/1/save \
  -H "Content-Type: application/json" \
  -d '{
    "result": {"email_sent": true},
    "error": ""
  }'
```

6. List all jobs in queue:
```bash
curl http://localhost:8080/jobs?queue=email
```
