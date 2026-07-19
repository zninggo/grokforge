.PHONY: backend migrate frontend build test tidy up down

export PATH := /usr/local/go/bin:$(PATH)

backend:
	cd backend && go build -o bin/grokforge ./cmd/grokforge
	cd backend && go build -o bin/grokforge-migrate ./cmd/grokforge-migrate

migrate: backend
	cd backend && ./bin/grokforge-migrate

frontend:
	cd frontend && pnpm install && pnpm build

build: backend frontend

tidy:
	cd backend && go mod tidy

test:
	cd backend && go test ./...

up:
	docker compose up -d

down:
	docker compose down
