DB_HOST = postgres
DB_PORT = 5432
DB_USER = postgres
DB_PASSWORD = password
DB_NAME = db_404chan

.PHONY: build run migrate seed

build:
	go build -buildvcs=false -o ./tmp/main .

run: build
	./tmp/main

migrate:
	go run ./cmd/migrate.go

seed:
	go run ./cmd/seed.go
