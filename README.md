# stock-watcher

Minimal Gin REST API for an in-memory stock watchlist with mock prices (±2% around known symbol bases).

## Run locally

```bash
export PORT=8080   # optional, default 8080
go run ./cmd/server
```

## API

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| GET | `/api/stocks` | List watched symbols with prices |
| GET | `/api/stocks/:symbol` | One symbol |
| POST | `/api/stocks` | Body: `{"symbol":"AAPL"}` |
| DELETE | `/api/stocks/:symbol` | Remove from watchlist |

## Docker

```bash
docker build -t stock-watcher .
docker run --rm -p 8080:8080 -e PORT=8080 stock-watcher
```
