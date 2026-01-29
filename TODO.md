9.  **Soft Deletes:** A `deleted_at` column support so data isn't permanently lost immediately.
10. **Webhooks / Events:** A system to notify external services when data changes (e.g., "Order Created" -> Trigger Email).
7.  **Batch Operations:** Bulk `Create`, `Update`, or `Delete` APIs. Creating 100 products one by one is slow.
8.  **File Uploads / Media Handling:** No visible support for `multipart/form-data` to upload images/files, which is mandatory for CMS/E-com.
6.  **Aggregation:** Endpoints for `count`, `sum`, `avg`, `min`, `max`. (Critical for dashboards/analytics).
5.  **Relations / Population:** Fetching related data (e.g., "Get Product **and** its Author"). Currently requires multiple API calls.

### Dockerise 

### Create a Docker Container (Optional)

You can also create a Dockerfile for containerized deployment:

```dockerfile
# Dockerfile
FROM golang:1.24 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o moon ./cmd/moon

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY --from=builder /app/moon .
COPY samples/moon.conf /etc/moon.conf

EXPOSE 6006
CMD ["./moon", "--config", "/etc/moon.conf"]
```

Build and run:

```bash
docker build -t moon:latest .
docker run -p 6006:6006 -v /etc/moon.conf:/etc/moon.conf moon:latest
```