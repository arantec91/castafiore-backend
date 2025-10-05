# Instrucciones de Configuración - Castafiore Backend

## 🚀 Configuración Inicial

### 1. Configurar Base de Datos

#### Opción A: Usar Docker (Recomendado)
```bash
# Iniciar PostgreSQL con Docker Compose
docker-compose up -d postgres

# La base de datos estará disponible en:
# Host: localhost
# Puerto: 5432
# Base de datos: castafiore
# Usuario: castafiore_user
# Contraseña: castafiore_password
```

#### Opción B: PostgreSQL Local
```sql
-- Crear base de datos y usuario
CREATE DATABASE castafiore;
CREATE USER castafiore_user WITH ENCRYPTED PASSWORD 'castafiore_password';
GRANT ALL PRIVILEGES ON DATABASE castafiore TO castafiore_user;
```

### 2. Configurar Variables de Entorno

Copia `.env.example` a `.env` y ajusta los valores:

```bash
PORT=8080
DATABASE_URL=postgres://castafiore_user:castafiore_password@localhost/castafiore?sslmode=disable
JWT_SECRET=tu-clave-secreta-super-segura-cambiar-en-produccion
MUSIC_PATH=C:\Music  # Ruta a tu biblioteca musical
MAX_CONCURRENT_STREAMS=3
MAX_DOWNLOADS_PER_DAY=50
```

### 3. Ejecutar Migraciones

#### Con Docker Compose activo:
```bash
# Ejecutar migraciones
psql -h localhost -U castafiore_user -d castafiore -f migrations/001_initial_schema.sql
psql -h localhost -U castafiore_user -d castafiore -f migrations/002_sample_data.sql
```

#### O usando pgAdmin/Adminer:
- Accede a http://localhost:8081 (Adminer)
- Conecta con las credenciales de PostgreSQL
- Ejecuta los archivos SQL de la carpeta `migrations/`

### 4. Crear Usuario Administrador

```bash
# Crear usuario admin
go run scripts/create-user.go admin admin@castafiore.local mypassword premium
```

### 5. Organizar Biblioteca Musical

Estructura recomendada:
```
C:\Music\
├── Artista 1\
│   ├── Álbum 1\
│   │   ├── 01 - Canción 1.mp3
│   │   ├── 02 - Canción 2.mp3
│   │   └── cover.jpg
│   └── Álbum 2\
└── Artista 2\
```

## 🎯 Inicio Rápido

### Método 1: Script de Windows
```bash
# Ejecutar script de inicio
start.bat
```

### Método 2: Comandos manuales
```bash
# Instalar dependencias
go mod download

# Compilar y ejecutar
go run cmd/server/main.go
```

### Método 3: Usando Makefile (requiere Make)
```bash
# Iniciar servicios Docker
make docker-up

# Ejecutar en modo desarrollo
make dev

# Crear usuario
make user ARGS="admin admin@test.com password123 premium"
```

## 🧪 Probar la API

### Endpoints básicos:
```bash
# Test de conectividad
curl "http://localhost:8080/rest/ping"

# Información de licencia
curl "http://localhost:8080/rest/getLicense"

# Health check
curl "http://localhost:8080/health"

# Info del servidor
curl "http://localhost:8080/"
```

### Con autenticación:
```bash
# Obtener carpetas de música (requiere usuario)
curl "http://localhost:8080/rest/getMusicFolders?u=admin&p=mypassword"

# Obtener índice de artistas
curl "http://localhost:8080/rest/getIndexes?u=admin&p=mypassword"
```

## 🔧 Desarrollo

### Estructura de archivos importantes:
- `cmd/server/main.go` - Punto de entrada
- `internal/api/routes.go` - Definición de rutas
- `internal/subsonic/` - Implementación API Subsonic
- `internal/auth/` - Sistema de autenticación
- `internal/database/` - Conexión y migraciones
- `migrations/` - Scripts SQL

### Comandos útiles:
```bash
# Ejecutar tests
go test ./...

# Formatear código
go fmt ./...

# Verificar dependencias
go mod tidy

# Compilar para producción
go build -o bin/castafiore.exe cmd/server/main.go
```

## 🌐 Clientes Compatibles

El servidor es compatible con clientes Subsonic como:
- **Castafiore** (tu app)
- Subsonic (oficial)
- DSub (Android)
- Sublime Music (Linux)
- Sonixd (multiplataforma)
- Navidrome Web UI

### Configuración en clientes:
- **URL del servidor**: `http://localhost:8080`
- **Endpoint**: `/rest`
- **Usuario**: El que creaste con el script
- **Contraseña**: La que configuraste

## 🐳 Docker

### Desarrollo completo con Docker:
```bash
# Iniciar todos los servicios
docker-compose up -d

# Ver logs
docker-compose logs -f

# Detener servicios
docker-compose down
```

### Servicios incluidos:
- **PostgreSQL**: Puerto 5432
- **Redis**: Puerto 6379 (para caché futuro)
- **Adminer**: Puerto 8081 (gestión de BD)

## 🔒 Seguridad

### Para producción:
1. Cambia `JWT_SECRET` por una clave fuerte
2. Usa HTTPS
3. Configura firewall adecuado
4. Usa contraseñas seguras para BD
5. Considera usar variables de entorno del sistema

### Limitaciones por plan:
- **Free**: 1 stream, 10 descargas/día
- **Pro**: 3 streams, 50 descargas/día  
- **Premium**: 5 streams, 100 descargas/día

## 📊 Monitoreo

### Endpoints de estado:
- `/health` - Estado general del servidor
- `/` - Información del servicio y versión

### Logs:
El servidor imprime logs en consola con información de:
- Inicio del servidor
- Conexiones de base de datos
- Errores de autenticación
- Requests de API

## 🚨 Solución de Problemas

### Error de conexión a base de datos:
1. Verifica que PostgreSQL esté corriendo
2. Confirma las credenciales en DATABASE_URL
3. Revisa que el puerto 5432 esté disponible

### Error de autenticación:
1. Verifica que el usuario existe en la BD
2. Confirma que la contraseña es correcta
3. Revisa los logs del servidor

### Error de permisos de archivos:
1. Verifica que MUSIC_PATH existe
2. Confirma permisos de lectura en la carpeta
3. Revisa que los archivos no estén bloqueados
