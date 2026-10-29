# ==============================================================================
# NORTABIL: Motor de Partidas Dobradas (Núcleo ERP Contábil)
# Dockerfile Multi-Stage de Alta Eficiência e Segurança
# ==============================================================================

# Estágio 1: Build da aplicação em Go
FROM golang:alpine AS builder

WORKDIR /app

# Instala dependências do sistema necessárias para compilação estática
RUN apk add --no-cache git ca-certificates tzdata

# Copia arquivos de dependências e baixa os módulos
COPY go.mod go.sum ./
RUN go mod download

# Copia todo o código-fonte da aplicação
COPY . .

# Compila o binário de forma 100% estática (CGO desabilitado) para máxima portabilidade
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/bin/nortabil-api \
    ./cmd/api

# Estágio 2: Imagem final de execução ultra-leve (Alpine)
FROM alpine:3.19

# Instala certificados SSL e configura timezone brasileiro (América/São Paulo)
RUN apk --no-cache add ca-certificates tzdata
ENV TZ=America/Sao_Paulo

WORKDIR /app

# Cria usuário não-root por boas práticas de segurança em contêineres
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

# Copia o binário compilado a partir do estágio anterior
COPY --from=builder /app/bin/nortabil-api /app/nortabil-api

# Expõe a porta padrão da API
EXPOSE 8080

# Comando de inicialização
ENTRYPOINT ["/app/nortabil-api"]

