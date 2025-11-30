DB_HOST = postgres
DB_PORT = 5432
DB_USER = postgres
DB_PASSWORD = password
DB_NAME = db_404chan

DB_URL = host=$(DB_HOST) user=$(DB_USER) password=$(DB_PASSWORD) dbname=$(DB_NAME) port=$(DB_PORT) sslmode=disable
MIGRATIONS_DIR = migrations
DRIVER = postgres

.PHONY: up down status new migrate-up migrate-down migrate-status create-migration

migrate-up:
	goose -dir $(MIGRATIONS_DIR) $(DRIVER) "$(DB_URL)" up

migrate-down:
	goose -dir $(MIGRATIONS_DIR) $(DRIVER) "$(DB_URL)" down

migrate-status:
	goose -dir $(MIGRATIONS_DIR) $(DRIVER) "$(DB_URL)" status

create-migration:
ifndef NAME
	$(error NAME is not set. Usage: make new NAME=add_users_table)
endif
	goose -dir $(MIGRATIONS_DIR) create $(NAME) sql

up: migrate-up
down: migrate-down
status: migrate-status
new: create-migration
