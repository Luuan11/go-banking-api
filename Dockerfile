FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /bin/api ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /bin/api /api

USER nonroot:nonroot
EXPOSE 8080

ENTRYPOINT ["/api"]
