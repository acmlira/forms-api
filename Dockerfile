# Etapa 1: builder
FROM golang:1.24 AS builder

WORKDIR /app

# Copia arquivos go.mod e go.sum antes (cache de dependências)
COPY go.mod go.sum ./
RUN go mod download

# Copia o restante da aplicação
COPY . .

# Gera a documentação Swagger
RUN go install github.com/swaggo/swag/cmd/swag@latest && swag init --generalInfo main.go --output ./docs

# Compila binário
RUN go build -o server ./main.go

# Porta da aplicação
EXPOSE 8080

# Executa o binário
CMD ["./server"]