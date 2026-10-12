FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /bin/api ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot

ARG VERSION=dev
LABEL org.opencontainers.image.title="bank-api" \
      org.opencontainers.image.source="https://github.com/Luuan11/go-banking-api" \
      org.opencontainers.image.version="${VERSION}"

COPY --from=builder /bin/api /api

USER nonroot:nonroot
EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=3 \
    CMD ["/api", "healthcheck"]

ENTRYPOINT ["/api"]
