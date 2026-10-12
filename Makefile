TEST_DATABASE_URL ?= postgres://bankuser:bankpass@localhost:5432/bankdb?sslmode=disable

.PHONY: run test test-integration cover lint vuln up up-observability down

run:
	go run ./cmd/api

test:
	go test -race ./...

test-integration:
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test -race -count=1 ./...

cover:
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test -count=1 -coverpkg=./... -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -n 1

lint:
	test -z "$$(gofmt -l .)"
	go vet ./...

vuln:
	go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...

up:
	docker compose up -d --build --wait

up-observability:
	docker compose --profile observability up -d --build --wait

down:
	docker compose --profile observability down
