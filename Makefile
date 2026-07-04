DB_URL ?= postgres://postgres:postgres@localhost:5432/realworld_dev?sslmode=disable

.PHONY: migrate-up migrate-down build

migrate-up:
	migrate -path migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path migrations -database "$(DB_URL)" down 1

build:
	go build -o bin/blog-api ./cmd/server