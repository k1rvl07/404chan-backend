DB_URL = host=localhost user=postgres_admin password=admin_password dbname=db_404chan port=5432 sslmode=disable
MIGRATIONS_DIR = migrations
DRIVER = postgres

migrate-up:
	goose -dir $(MIGRATIONS_DIR) $(DRIVER) "$(DB_URL)" up

migrate-down:
	goose -dir $(MIGRATIONS_DIR) $(DRIVER) "$(DB_URL)" down

migrate-status:
	goose -dir $(MIGRATIONS_DIR) $(DRIVER) "$(DB_URL)" status

create-migration:
	goose -dir $(MIGRATIONS_DIR) create $(NAME) sql

up: migrate-up
down: migrate-down
status: migrate-status
new: create-migration
