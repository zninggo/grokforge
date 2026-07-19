.PHONY: backend migrate frontend frontend-embed build test tidy up down

export PATH := /usr/local/go/bin:$(PATH)
export GOTOOLCHAIN ?= local

backend:
	cd backend && go build -o bin/grokforge ./cmd/grokforge
	cd backend && go build -o bin/grokforge-migrate ./cmd/grokforge-migrate

migrate: backend
	cd backend && ./bin/grokforge-migrate

frontend:
	cd frontend && pnpm install && pnpm build

frontend-embed: frontend
	rm -rf backend/internal/web/dist
	mkdir -p backend/internal/web/dist
	cp -a frontend/out/. backend/internal/web/dist/

build: frontend-embed backend

tidy:
	cd backend && go mod tidy

test:
	cd backend && go test ./...

up:
	docker compose up -d

down:
	docker compose down
