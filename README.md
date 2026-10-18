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

## API

### Create a task

```bash
curl -i -X POST http://localhost:8080/tasks \
  -H 'Content-Type: application/json' \
  -d '{"title":"Write assessment","description":"Create the first PR"}'
```

### List tasks

```bash
curl -i http://localhost:8080/tasks
```

### Get one task

```bash
curl -i http://localhost:8080/tasks/1
```

### Update a task

```bash
curl -i -X PUT http://localhost:8080/tasks/1 \
  -H 'Content-Type: application/json' \
  -d '{"title":"Write assessment","description":"Review Codzee comments","status":"in_progress"}'
```

### Delete a task

```bash
curl -i -X DELETE http://localhost:8080/tasks/1
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
