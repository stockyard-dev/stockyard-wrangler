# Stockyard Wrangler

**Job queue.** Enqueue jobs, dispatch via HTTP callback, retry with exponential backoff. Single binary, no Redis, no external dependencies.

Part of the [Stockyard](https://stockyard.dev) suite of self-hosted developer tools.

## Quick Start

```bash
curl -sfL https://stockyard.dev/install/wrangler | sh
wrangler

# Or with Docker
docker run -p 8810:8810 -v wrangler-data:/data ghcr.io/stockyard-dev/stockyard-wrangler:latest
```

Dashboard at [http://localhost:8810/ui](http://localhost:8810/ui)

## How It Works

Wrangler uses HTTP callbacks — enqueue a job with a payload and a callback URL, and Wrangler POSTs to that URL when the job is ready. No code execution inside Wrangler, just HTTP dispatch.

```bash
# 1. Create a queue
curl -X POST http://localhost:8810/api/queues \
  -H "Content-Type: application/json" \
  -d '{"name":"emails"}'

# 2. Enqueue a job
curl -X POST http://localhost:8810/api/queues/{queue_id}/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "callback_url": "http://localhost:3000/workers/send-email",
    "payload": {"user_id": 123, "template": "welcome"},
    "max_attempts": 3,
    "backoff_seconds": 60
  }'

# Wrangler POSTs the payload to your callback URL
# 2xx = success, anything else = retry with backoff
```

## API

| Method | Path | Description |
|--------|------|-------------|
| POST | /api/queues | Create queue |
| GET | /api/queues | List queues with counts |
| GET | /api/queues/{id} | Queue detail |
| DELETE | /api/queues/{id} | Delete queue + all jobs |
| POST | /api/queues/{id}/jobs | Enqueue job |
| GET | /api/queues/{id}/jobs | List jobs (filter by ?status=) |
| GET | /api/queues/{id}/stats | Queue depth and counts |
| GET | /api/jobs/{id} | Job detail |
| DELETE | /api/jobs/{id} | Cancel job |
| POST | /api/jobs/{id}/retry | Retry dead/failed job |
| GET | /api/dlq | Dead letter queue |
| GET | /api/status | Service stats |
| GET | /health | Health check |
| GET | /ui | Web dashboard |

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| PORT | 8810 | HTTP port |
| DATA_DIR | ./data | SQLite data directory |
| RETENTION_DAYS | 30 | Completed job retention |
| WRANGLER_LICENSE_KEY | | Pro license key |

## Free vs Pro

| Feature | Free | Pro ($4.99/mo) |
|---------|------|----------------|
| Queues | 1 | Unlimited |
| Jobs/month | 1,000 | Unlimited |
| Max retries | 3 | Unlimited |
| Scheduled jobs (run_at) | — | ✓ |
| Priority jobs | — | ✓ |
| Job history | 7 days | 90 days |
| Webhook on failure | — | ✓ |

## License

Apache 2.0 — see [LICENSE](LICENSE).
