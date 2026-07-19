.PHONY: backend frontend build test tidy

export PATH := /usr/local/go/bin:$(PATH)

backend:
	cd backend && go build -o bin/grokforge ./cmd/grokforge

frontend:
	cd frontend && pnpm install && pnpm build

build: backend frontend

tidy:
	cd backend && go mod tidy

test:
	cd backend && go test ./...
