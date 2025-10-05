# Makefile para Castafiore Backend

.PHONY: help build run test clean dev docker-up docker-down migrate

# Variables
BINARY_NAME=castafiore
BUILD_DIR=bin
SOURCE_DIR=cmd/server

# Ayuda
help:
	@echo "Comandos disponibles:"
	@echo "  build       - Compilar el proyecto"
	@echo "  run         - Ejecutar el servidor"
	@echo "  test        - Ejecutar tests"
	@echo "  clean       - Limpiar archivos generados"
	@echo "  dev         - Modo desarrollo con recarga automática"
	@echo "  docker-up   - Iniciar servicios Docker"
	@echo "  docker-down - Detener servicios Docker"
	@echo "  migrate     - Ejecutar migraciones"
	@echo "  user        - Crear usuario (requiere ARGS)"

# Compilar el proyecto
build:
	@echo "🔨 Compilando $(BINARY_NAME)..."
	go build -o $(BUILD_DIR)/$(BINARY_NAME).exe $(SOURCE_DIR)/main.go

# Ejecutar el servidor
run: build
	@echo "🚀 Iniciando servidor..."
	./$(BUILD_DIR)/$(BINARY_NAME).exe

# Ejecutar tests
test:
	@echo "🧪 Ejecutando tests..."
	go test -v ./...

# Limpiar archivos generados
clean:
	@echo "🧹 Limpiando archivos generados..."
	go clean
	rm -rf $(BUILD_DIR)

# Modo desarrollo
dev:
	@echo "💻 Iniciando en modo desarrollo..."
	go run $(SOURCE_DIR)/main.go

# Iniciar servicios Docker
docker-up:
	@echo "🐳 Iniciando servicios Docker..."
	docker-compose up -d

# Detener servicios Docker
docker-down:
	@echo "🐳 Deteniendo servicios Docker..."
	docker-compose down

# Ejecutar migraciones manualmente
migrate:
	@echo "📊 Ejecutando migraciones..."
	@echo "Conectando a la base de datos para ejecutar migraciones..."
	@psql -h localhost -U castafiore_user -d castafiore -f migrations/001_initial_schema.sql
	@psql -h localhost -U castafiore_user -d castafiore -f migrations/002_sample_data.sql

# Crear usuario (ejemplo: make user ARGS="admin admin@test.com password123 premium")
user:
	@echo "👤 Creando usuario..."
	go run scripts/create-user.go $(ARGS)

# Instalar dependencias
deps:
	@echo "📦 Instalando dependencias..."
	go mod download
	go mod tidy

# Formatear código
fmt:
	@echo "✨ Formateando código..."
	go fmt ./...

# Ejecutar linter
lint:
	@echo "🔍 Ejecutando linter..."
	golangci-lint run

# Generar documentación
docs:
	@echo "📚 Generando documentación..."
	godoc -http=:6060

# Versión
version:
	@echo "Castafiore Backend v1.0.0"
