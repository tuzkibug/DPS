.PHONY: run test build clean

run:
	cd backend && go run ./cmd/server

test:
	cd backend && go test ./...

test-verbose:
	cd backend && go test ./... -v

test-frontend:
	cd web-ui && npm test

build:
	cd backend && CGO_ENABLED=1 go build -o server ./cmd/server

build-docker:
	docker compose build

clean:
	rm -f backend/server backend/data.db

.PHONY: vet lint
vet:
	cd backend && go vet ./...

lint:
	cd web-ui && npx tsc --noEmit
