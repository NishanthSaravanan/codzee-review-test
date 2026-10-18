# Codzee Review Test — TaskFlow API

A small Go REST API for managing tasks. This project is intended to provide a clean baseline for evaluating AI code-review tools.

## Requirements

- Go 1.22 or newer

## Run

```bash
go run ./cmd
```

The server listens on `http://localhost:8080`.

## Test

```bash
go test ./...
go test -race ./...
```

## Authentication

Set `TASKFLOW_USERNAME` and `TASKFLOW_PASSWORD` in the server environment. Obtain a token from `/login`, then send it as a Bearer token with task requests. Tokens expire after one hour.

```bash
curl -i -X POST http://localhost:8080/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"<username>","password":"<password>"}'
```

## API

### Create a task

```bash
curl -i -X POST http://localhost:8080/tasks \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <access-token>' \
  -d '{"title":"Write assessment","description":"Create the first PR"}'
```

### List tasks

```bash
curl -i http://localhost:8080/tasks \
  -H 'Authorization: Bearer <access-token>'
```

### Get one task

```bash
curl -i http://localhost:8080/tasks/1 \
  -H 'Authorization: Bearer <access-token>'
```

### Update a task

```bash
curl -i -X PUT http://localhost:8080/tasks/1 \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <access-token>' \
  -d '{"title":"Write assessment","description":"Review Codzee comments","status":"in_progress"}'
```

### Delete a task

```bash
curl -i -X DELETE http://localhost:8080/tasks/1 \
  -H 'Authorization: Bearer <access-token>'
```

## Task model

- `id`: server-generated positive integer
- `title`: required, trimmed, 1–120 characters
- `description`: optional, up to 2,000 characters
- `status`: `pending`, `in_progress`, or `completed`
- `created_at`: UTC timestamp

## Notes

- The initial version uses an in-memory store; data is lost when the process stops.
- The API intentionally starts with a small, maintainable baseline. Add features on separate branches for the assessment.
- Do not commit credentials, real tokens, or production data.
