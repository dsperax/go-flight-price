# Flight Price Service

A Go microservice that aggregates flight prices from multiple providers concurrently, identifies the cheapest and fastest options, and exposes a secure RESTful API with JWT authentication.

## What does this API do?

You send an origin airport, a destination airport, and a date — the service fires parallel requests to three flight providers (Amadeus, Skyscanner, CheapFlights), collects all available flights, and returns a single structured response containing:

- **All flights** from every provider, sorted by price (ascending)
- **The cheapest flight** — lowest price, duration as tiebreaker
- **The fastest flight** — shortest duration, price as tiebreaker
- **Provider errors**, if any provider failed individually (partial success still returns HTTP 200)

Results are cached for 30 seconds, so back-to-back requests for the same route skip the provider calls and return instantly with `"cached": true`.

Beyond the main search, the API also offers a **historical price trend** (24 months of monthly averages) and a **live SSE stream** that pushes updated prices to connected clients every 30 seconds.

All endpoints except `/health` and `/auth/login` are protected by JWT. You log in once, get a Bearer token, and include it in subsequent requests.

## Running locally

**Requirements:** Go ≥ 1.24 and Make.

```bash
# 1. Clone the repository
git clone <repo-url>
cd go-flight-price

# 2. Start the server (sets up .env automatically)
make dev
```

The server starts on **http://localhost:8080**.

To verify it is running:
```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

### Getting a token

Every protected endpoint requires a JWT Bearer token. Request one via `POST /auth/login`:

```bash
curl -s -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'
```

Response:
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 3600
}
```

Copy the `access_token` value and pass it in the `Authorization` header of all subsequent requests:

```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

### Making your first search

```bash
# Store the token in a shell variable for convenience
TOKEN=$(curl -s -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' \
  | grep -o '"access_token":"[^"]*' | cut -d'"' -f4)

# Search flights
curl "http://localhost:8080/flights/search?origin=GRU&destination=JFK&date=2026-06-10" \
  -H "Authorization: Bearer $TOKEN"
```

## Features

| Category | Detail |
|---|---|
| **Search** | Aggregates results from 3 providers in parallel; returns cheapest and fastest flights |
| **Auth** | JWT HS256 — obtain a token via `POST /auth/login`, pass as `Authorization: Bearer <token>` |
| **Caching** | In-memory TTL cache (default 30 s) avoids redundant provider calls |
| **History** | 24 months of monthly average prices for any route |
| **SSE** | Live price stream via Server-Sent Events, updated every 30 s |
| **Docker** | Multi-stage build produces a minimal `scratch`-based image |
| **Testing** | 90 unit + integration tests across all packages |

## Environment variables

Copy `.env.example` to `.env` and adjust as needed. **Never commit `.env`.**

| Variable | Default | Description |
|---|---|---|
| `APP_PORT` | `8080` | HTTP listen port |
| `APP_ENV` | `development` | Runtime environment label |
| `JWT_SECRET` | `changeme` | HMAC-SHA256 signing key — **change in production** |
| `JWT_EXPIRATION_MINUTES` | `60` | Token lifetime in minutes |
| `AUTH_USERNAME` | `admin` | Login username |
| `AUTH_PASSWORD` | `admin123` | Login password — **change in production** |
| `PROVIDER_TIMEOUT_SECONDS` | `3` | Per-provider request deadline |
| `CACHE_TTL_SECONDS` | `30` | How long search results are cached |
| `AMADEUS_API_KEY` | _(empty)_ | Amadeus provider key (mock — not used yet) |
| `SKYSCANNER_API_KEY` | _(empty)_ | Skyscanner provider key (mock — not used yet) |
| `CHEAPFLIGHTS_API_KEY` | _(empty)_ | CheapFlights provider key (mock — not used yet) |

## API reference

All endpoints except `/health` and `POST /auth/login` require a valid JWT:

```
Authorization: Bearer <token>
```

### `GET /health`

Health check. No authentication required.

**Response `200`**
```json
{ "status": "ok" }
```

---

### `POST /auth/login`

Obtain a JWT token.

**Request body**
```json
{ "username": "admin", "password": "admin123" }
```

**Response `200`**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 3600
}
```

**Response `401`**
```json
{ "error": "unauthorized", "message": "invalid credentials" }
```

---

### `GET /flights/search`

Search for flights between two airports. Queries all providers concurrently.

**Query parameters**

| Param | Required | Format | Example |
|---|---|---|---|
| `origin` | ✅ | IATA code (case-insensitive) | `GRU` |
| `destination` | ✅ | IATA code (case-insensitive) | `JFK` |
| `date` | ✅ | `YYYY-MM-DD` | `2026-06-10` |

**Response `200`**
```json
{
  "origin": "GRU",
  "destination": "JFK",
  "date": "2026-06-10",
  "cheapest_flight": { "..." },
  "fastest_flight":  { "..." },
  "flights": [
    {
      "provider": "Amadeus",
      "airline": "Gol",
      "flight_number": "G3GRUJFK",
      "origin": "GRU",
      "destination": "JFK",
      "departure_time": "2026-06-10T18:30:00Z",
      "arrival_time": "2026-06-11T05:00:00Z",
      "duration_minutes": 650,
      "price": 2980.00,
      "currency": "BRL"
    }
  ],
  "provider_errors": [],
  "cached": false
}
```

`flights` is sorted by price ascending (duration as tiebreaker).  
`cached: true` on subsequent calls within the TTL window.

**Response `400`** — missing or invalid parameters  
**Response `401`** — missing or invalid token  
**Response `502`** — all providers failed (includes per-provider error details)

---

### `GET /flights/history`

Returns 24 months of monthly average prices for a route (last 2 years).

**Query parameters**

| Param | Required |
|---|---|
| `origin` | ✅ |
| `destination` | ✅ |

**Response `200`**
```json
{
  "origin": "GRU",
  "destination": "JFK",
  "history": [
    { "year": 2024, "month": 5,  "avg_price": 3245.00, "currency": "BRL" },
    { "year": 2024, "month": 6,  "avg_price": 3569.50, "currency": "BRL" },
    { "year": 2026, "month": 4,  "avg_price": 2920.50, "currency": "BRL" }
  ]
}
```

---

### `GET /subscribe/{route}`

Opens a Server-Sent Events stream. The server emits one event immediately on connect, then every **30 seconds** via `time.Ticker`.

**Path parameter**

| Param | Format | Example |
|---|---|---|
| `route` | `ORIGIN-DESTINATION` (case-insensitive) | `GRU-JFK` |

**Query parameter (optional)**

| Param | Format | Default |
|---|---|---|
| `date` | `YYYY-MM-DD` | tomorrow |

**Response headers**
```
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
```

**Event stream example**
```
data: {"origin":"GRU","destination":"JFK","date":"2026-06-11","cheapest_flight":{...},...}

data: {"origin":"GRU","destination":"JFK","date":"2026-06-11","cheapest_flight":{...},...}
```

Each `data:` line is a complete JSON payload (same shape as `/flights/search`). Disconnect by closing the HTTP connection.

**Response `400`** — route not in `ORIGIN-DESTINATION` format  
**Response `401`** — missing or invalid token

## Error response shape

```json
{
  "error": "error_code",
  "message": "human-readable description",
  "details": [...]
}
```

| Code | HTTP | Meaning |
|---|---|---|
| `invalid_request` | 400 | Validation failure |
| `unauthorized` | 401 | Missing or invalid JWT |
| `not_found` | 404 | Route not found |
| `provider_unavailable` | 502 | All flight providers failed |
| `internal_error` | 500 | Unexpected server error |

## Running tests

```bash
make test          # run all tests (no cache)
make test-v        # verbose output
make test-cover    # generate coverage.out + open HTML report
```

Current status: **90 tests, all passing** across 7 packages.

## Makefile targets

```
make dev           # start dev server (copies .env, tidy, go run)
make run           # build binary and run it
make build         # compile to bin/flight-price-service
make test          # run all tests
make test-v        # run tests, verbose
make test-cover    # run tests with coverage report
make docker-build  # build Docker image
make docker-run    # build and start container in background
make docker-stop   # stop running container
make docker-logs   # tail container logs
make tidy          # go mod tidy + verify
make help          # list all targets
```

## Docker

```bash
# Build and run
make docker-run

# Check logs
make docker-logs

# Stop
make docker-stop
```

The Dockerfile uses a two-stage build:
1. **Builder** (`golang:1.24-alpine`) — compiles a statically linked binary with `CGO_ENABLED=0`.
2. **Runtime** (`scratch`) — copies only the binary and CA certificates; final image is ~10 MB.

## Project structure

```
.
├── cmd/api/main.go               # entry point
├── internal/
│   ├── auth/                     # JWT generation, login handler, Bearer middleware
│   ├── cache/                    # generic thread-safe TTL cache
│   ├── config/                   # environment variable loading
│   ├── flights/                  # domain models, service (concurrency), handler, errors
│   ├── history/                  # historical price endpoint
│   ├── httpx/                    # standardised JSON response helpers
│   ├── providers/                # mock adapters: Amadeus, Skyscanner, CheapFlights
│   ├── server/                   # chi router wiring (testable)
│   └── sse/                      # Server-Sent Events handler
├── postman/                      # Postman collection (21 requests, 5 folders)
├── Dockerfile
├── .env.example
├── Makefile
└── go.mod
```

## Architecture decisions

**Concurrency model** — `flights.Service.Search` spawns one goroutine per provider guarded by a `sync.WaitGroup`. Each goroutine runs with an independent context deadline (`PROVIDER_TIMEOUT_SECONDS`). Results are collected through a buffered channel. A `defer recover()` in every goroutine prevents a panicking provider from crashing the server — it is converted into a `ProviderError` instead.

**Typed errors** — `ValidationError` (field-aware) and `AllProvidersFailedError` (carries each provider's individual failure) allow the HTTP handler to dispatch responses precisely with `errors.As`, without string matching.

**Provider interface** — `FlightProvider { Name() string; Search(ctx, req) ([]Flight, error) }` decouples the service from concrete adapters. The same interface is used by `sse.Searcher`, keeping the SSE handler independently testable.

**Cache** — generic `Cache[K, V]` backed by `sync.RWMutex`. Cache key is `origin:destination:date`. The `cached` boolean field in the response tells the client whether the data came from cache.

**Server as `http.Handler`** — `server.New(cfg, providers)` returns an `http.Handler` rather than owning a `*http.Server`. This makes integration tests straightforward with `httptest.NewServer`.

## HTTPS / TLS in production

The service itself speaks plain HTTP. TLS termination should be handled at the infrastructure layer.

**Option 1 — Reverse proxy (recommended)**  
Place the service behind Nginx, Caddy, or a cloud load balancer. Caddy handles certificate renewal automatically:

```
flight.example.com {
    reverse_proxy localhost:8080
}
```

**Option 2 — Native TLS in Go**  
Replace `http.ListenAndServe` with `http.ListenAndServeTLS` in `cmd/api/main.go`:

```go
err = http.ListenAndServeTLS(
    ":443",
    "/etc/ssl/certs/server.crt",
    "/etc/ssl/private/server.key",
    handler,
)
```

Certificates can be obtained from **Let's Encrypt** via `golang.org/x/crypto/acme/autocert`.

**Minimum TLS configuration checklist:**
- TLS 1.2 as minimum version (`tls.VersionTLS12`)
- Strong cipher suites only (disable RC4, 3DES, export ciphers)
- `Strict-Transport-Security: max-age=63072000; includeSubDomains` response header
- Rotate `JWT_SECRET` and credentials via environment variables or a secrets manager (HashiCorp Vault, AWS Secrets Manager, etc.)

## Security notes

- `JWT_SECRET`, `AUTH_USERNAME`, and `AUTH_PASSWORD` are read exclusively from environment variables — never hard-coded.
- `.env` is listed in `.gitignore` and never committed.
- All authenticated endpoints validate the Bearer token on every request via middleware before routing.
- Provider API keys are isolated in the `Config` struct and never logged or returned in responses.

