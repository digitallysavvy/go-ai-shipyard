.PHONY: setup dev server web test reset

## setup: install web dependencies and create .env from the template
setup:
	@test -f .env || cp .env.example .env
	go mod download
	cd web && pnpm install

## dev: run the Go server (:8080) and the web app (:3000) together
dev:
	@trap 'kill 0' EXIT; \
	go run ./server & \
	(cd web && pnpm dev) & \
	wait

## server: run only the Go server
server:
	go run ./server

## web: run only the web app
web:
	cd web && pnpm dev

## test: vet and test the server, type-check the web app
test:
	go vet ./server
	go test -race ./server
	cd web && pnpm typecheck

## reset: delete the demo workspace and coding-agent sandboxes
reset:
	rm -rf .data
