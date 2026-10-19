# Comandos de desenvolvimento. Funcionam em macOS, Linux e Windows
# (via Git Bash / WSL), pois só dependem de docker, go e npm.

.DEFAULT_GOAL := help

## db-up: sobe o Postgres via docker-compose e espera ficar pronto
db-up:
	docker compose up -d --wait

## db-down: derruba o Postgres (mantém o volume de dados)
db-down:
	docker compose down

## migrate: aplica as migrations pendentes
migrate:
	go run ./cmd/server migrate up

## migrate-status: mostra quais migrations já rodaram
migrate-status:
	go run ./cmd/server migrate status

## run: sobe o backend em localhost:8080
run:
	go run ./cmd/server run --address=localhost:8080

## openapi: gera a spec OpenAPI 3 a partir do código Go
openapi:
	go run ./cmd/server openapi > api/openapi.yaml
	@echo "spec escrita em api/openapi.yaml"

## client: gera o client TypeScript a partir da spec
client: openapi
	cd web && npm run generate:client

## dev: sobe banco, migra e roda backend + frontend
dev: db-up migrate
	@echo "Rode em terminais separados: make run  |  cd web && npm run dev"

## help: lista os comandos
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'

.PHONY: db-up db-down migrate migrate-status run openapi client dev help
