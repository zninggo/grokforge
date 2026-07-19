.PHONY: backend migrate frontend frontend-embed build test tidy up down redis-dev check-no-chat image

export PATH := /usr/local/go/bin:$(PATH)
export GOTOOLCHAIN ?= local
IMAGE ?= ghcr.io/zninggo/grokforge
TAG ?= latest

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
	$(MAKE) check-no-chat

check-no-chat:
	bash scripts/check_no_chat.sh

# Optional isolated data sandbox (not default daily path).
up:
	docker compose up -d

down:
	docker compose down

# Optional Redis only for local job queue (loopback). Does not start Postgres.
redis-dev:
	docker run --rm -d --name grokforge-redis-dev \
		-p 127.0.0.1:26379:6379 \
		redis:7-alpine redis-server --save "" --appendonly no

# Production/release image build — not for day-to-day development.
image:
	docker build -t $(IMAGE):$(TAG) .
