# ==============================================================================
# Makefile para o Projeto Nortabil (Motor de Partidas Dobradas)
# ==============================================================================

.PHONY: help build run test test-coverage seed docker-up docker-down docker-logs clean

help:
	@echo "Comandos disponíveis:"
	@echo "  make build         - Compila o binário da aplicação"
	@echo "  make run           - Executa a aplicação localmente"
	@echo "  make seed          - Executa a carga demonstrativa com dados do Norte de Minas"
	@echo "  make test          - Executa todos os testes unitários e de concorrência"
	@echo "  make test-coverage - Executa os testes e gera relatório de cobertura HTML"
	@echo "  make docker-up     - Sobe todo o ambiente conteinerizado (API + PostgreSQL)"
	@echo "  make docker-down   - Encerra os contêineres Docker"
	@echo "  make docker-logs   - Exibe os logs dos contêineres"
	@echo "  make clean         - Limpa binários e arquivos temporários de build"

build:
	go build -ldflags="-w -s" -o bin/nortabil-api ./cmd/api

run:
	go run ./cmd/api

seed:
	go run ./cmd/seed/main.go

test:
	go test -v ./tests/...

test-coverage:
	go test -coverprofile=coverage.out ./tests/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Relatório gerado em coverage.html"

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f

clean:
	rm -rf bin/ coverage.out coverage.html

