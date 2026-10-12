# Bank System API

A banking REST API written in Go, following Clean Architecture. It handles customers, checking and savings accounts, deposits, withdrawals, transfers and account statements on top of PostgreSQL. Every money movement runs inside a database transaction with row-level locking.

## Features

- Customer registration with CPF validation (check digits), email validation and a minimum age of 18
- One checking and one savings account per customer
- Deposits, withdrawals and transfers that are atomic and safe under concurrent requests
- A daily debit limit per account (withdrawals + outgoing transfers)
- Paginated account statements, backed by an immutable ledger (`balance_before` / `balance_after`)
- Versioned SQL migrations embedded in the binary
- Structured JSON logs with request IDs, Prometheus metrics, graceful shutdown and server timeouts
- Unit tests, PostgreSQL integration tests (including concurrency tests) and CI

## Tech stack

Go 1.26 (standard library `net/http` router) · PostgreSQL 16 · `lib/pq` · Docker · GitHub Actions

## Project structure

```
cmd/api                       # entrypoint: wiring, HTTP server, graceful shutdown
internal/domain               # entities and business rules (Account, Customer, Transaction, CPF)
internal/usecase              # application use cases
internal/repository           # repository and unit-of-work interfaces
internal/repository/postgres  # PostgreSQL implementations
internal/handler/http         # HTTP handlers, routing and error mapping
internal/middleware           # logging, panic recovery, CORS
pkg/config                    # environment configuration
pkg/database                  # connection and embedded migrations
test/integration              # tests against a real PostgreSQL
```

## Running

### With Docker

```bash
docker compose up --build
```

The API listens on `http://localhost:8080`. Migrations run automatically on startup.

> If you ran an older version of this project before, its tables were created without migrations. Reset the volume once with `docker compose down -v`.

### Locally

```bash
docker compose up -d postgres
go run ./cmd/api
```

| Variable           | Default                                                                  |
| ------------------ | ------------------------------------------------------------------------ |
| `DATABASE_URL`     | `postgres://bankuser:bankpass@localhost:5432/bankdb?sslmode=disable`     |
| `SERVER_PORT`      | `8080`                                                                   |
| `SHUTDOWN_TIMEOUT` | `10s`                                                                    |
| `LOG_LEVEL`        | `info` (`debug`, `info`, `warn`, `error`)                                |
| `LOG_FORMAT`       | `json` (`json`, `text`)                                                  |

## API

All amounts are integers in **cents** (`1050` = R$ 10,50). Every response uses the same envelope:

```json
{ "success": true, "data": { } }
{ "success": false, "error": { "code": "INSUFFICIENT_BALANCE", "message": "insufficient balance" } }
```

| Method | Path                                 | Description                     |
| ------ | ------------------------------------ | ------------------------------- |
| GET    | `/health`                            | Health check (pings the database) |
| GET    | `/metrics`                           | Prometheus metrics              |
| POST   | `/api/customers`                     | Create a customer               |
| GET    | `/api/customers/{id}`                | Get a customer                  |
| POST   | `/api/accounts`                      | Open an account                 |
| GET    | `/api/accounts/{id}`                 | Get an account and its balance  |
| GET    | `/api/accounts/{id}/transactions`    | Statement (`?limit=20&offset=0`, newest first) |
| POST   | `/api/accounts/{id}/deposits`        | Deposit                         |
| POST   | `/api/accounts/{id}/withdrawals`     | Withdraw                        |
| POST   | `/api/transfers`                     | Transfer between accounts       |

### Example

```bash
# create a customer
curl -X POST localhost:8080/api/customers -d '{
  "full_name": "Maria Silva",
  "email": "maria@example.com",
  "cpf": "529.982.247-25",
  "phone": "11999999999",
  "birth_date": "1995-04-20"
}'

# open an account
curl -X POST localhost:8080/api/accounts -d '{"customer_id": "<customer id>", "type": "checking"}'

# deposit R$ 100,00
curl -X POST localhost:8080/api/accounts/<account id>/deposits -d '{"amount": 10000, "description": "salary"}'

# withdraw R$ 20,00
curl -X POST localhost:8080/api/accounts/<account id>/withdrawals -d '{"amount": 2000}'

# transfer R$ 30,00
curl -X POST localhost:8080/api/transfers -d '{
  "from_account_id": "<account id>",
  "to_account_id": "<other account id>",
  "amount": 3000,
  "description": "rent"
}'

# statement
curl "localhost:8080/api/accounts/<account id>/transactions?limit=10"
```

### Error codes

| Status | Codes |
| ------ | ----- |
| 400 | `INVALID_REQUEST`, `INVALID_FULL_NAME`, `INVALID_EMAIL`, `INVALID_CPF`, `INVALID_BIRTH_DATE`, `INVALID_ACCOUNT_TYPE`, `INVALID_CUSTOMER_ID`, `INVALID_ACCOUNT_ID`, `INVALID_AMOUNT`, `INVALID_DESCRIPTION`, `SAME_ACCOUNT_TRANSFER`, `INVALID_LIMIT`, `INVALID_OFFSET` |
| 404 | `CUSTOMER_NOT_FOUND`, `ACCOUNT_NOT_FOUND`, `ROUTE_NOT_FOUND` |
| 405 | `METHOD_NOT_ALLOWED` |
| 409 | `CUSTOMER_EXISTS`, `ACCOUNT_EXISTS` |
| 422 | `INSUFFICIENT_BALANCE`, `DAILY_LIMIT_EXCEEDED`, `ACCOUNT_INACTIVE` |
| 503 | `DATABASE_UNAVAILABLE` |

## Observability

### Logs

Logs are structured JSON on stdout. Every request gets an `X-Request-ID`: a well-formed incoming one is reused, otherwise one is generated. It is returned in the response and added to **every** log line written while handling the request, including errors and recovered panics.

```json
{"time":"...","level":"WARN","msg":"request","service":"bank-api","version":"1.2.0","method":"POST","route":"/api/accounts/{id}/withdrawals","path":"/api/accounts/0735.../withdrawals","status":422,"bytes":98,"duration_ms":3.412,"remote_addr":"10.0.0.7:51234","user_agent":"curl/8.5.0","error_code":"INSUFFICIENT_BALANCE","request_id":"7c1e..."}
```

- The level follows the status: `ERROR` for 5xx, `WARN` for 4xx and `INFO` otherwise. Health checks and metric scrapes log at `DEBUG`, so they don't flood the logs.
- `route` is the matched pattern, which is safe to group by. `path` is the concrete URL.
- `error_code` is the same code returned to the client.

### Metrics

`GET /metrics` exposes Prometheus metrics:

| Metric | Labels | Use |
| ------ | ------ | --- |
| `http_requests_total` | `method`, `route`, `status` | Traffic and error rate |
| `http_request_duration_seconds` | `method`, `route` | Latency (p50/p95/p99) |
| `http_requests_in_flight` | | Saturation |
| `bank_operations_total` | `operation`, `result` | Deposits, withdrawals and transfers, by `ok` or error code |
| `bank_operation_amount_cents_total` | `operation` | Money moved by successful operations |
| `go_sql_*` | `db_name` | Connection pool (open, in use, wait count and duration) |
| `bank_build_info` | `version`, `go_version` | Which version is running |
| `go_*`, `process_*` | | Runtime: goroutines, GC, memory, CPU, file descriptors |

`route` is the route pattern and the methods are a fixed list, so label cardinality stays bounded.

Example queries:

```promql
# error rate (5xx) per route
sum by (route) (rate(http_requests_total{status=~"5.."}[5m]))
  / sum by (route) (rate(http_requests_total[5m]))

# p95 latency per route
histogram_quantile(0.95, sum by (le, route) (rate(http_request_duration_seconds_bucket[5m])))

# transfers rejected for lack of funds
rate(bank_operations_total{operation="transfer", result="INSUFFICIENT_BALANCE"}[5m])
```

## Design notes

- **Consistency:** a money movement locks the involved account rows with `SELECT ... FOR UPDATE` inside a transaction. The balance update and the ledger entry are committed together. A `CHECK (balance >= 0)` constraint is the last line of defense.
- **No deadlocks on transfers:** both accounts are always locked in the same order (by id), so A→B and B→A running at the same time can't wait on each other.
- **Daily limit:** the sum of today's withdrawals and outgoing transfers (UTC day) can't go over `daily_limit` (R$ 1.000,00 by default). The row lock also makes this check race-free.
- **Race-safe uniqueness:** duplicate customers and accounts are rejected by unique constraints, and their violations are mapped to `409` responses, so concurrent requests don't turn into `500`s.

## Tests

```bash
# unit tests
go test ./...

# also run the integration tests (they TRUNCATE the target database)
docker compose up -d postgres
TEST_DATABASE_URL="postgres://bankuser:bankpass@localhost:5432/bankdb?sslmode=disable" go test -race ./...
```

The integration tests include concurrency scenarios: 20 parallel withdrawals against a balance that only covers 10 of them, and 20 parallel transfers going both ways between two accounts.

## Roadmap

- Authentication (JWT) and ownership checks
- Idempotency keys for money movements
- Account blocking / closing endpoints
- OpenAPI documentation
