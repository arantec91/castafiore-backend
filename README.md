# Castafiore Backend

Un servidor de streaming musical en Go con compatibilidad con la API Subsonic, diseñado para ofrecer características avanzadas como limitación de conexiones simultáneas, control de descargas y gestión de planes de usuarios.

## 🎵 Características

- **API compatible con Subsonic**: Integración completa con clientes existentes
- **Autenticación JWT**: Sistema seguro de autenticación
- **Gestión de planes**: Sistema de suscripciones con diferentes niveles
- **Control de concurrencia**: Limitación de conexiones simultáneas por usuario
- **Seguimiento de descargas**: Límites configurables de descargas diarias
- **Escaneo automático**: Organización inteligente de bibliotecas musicales
- **Streaming adaptativo**: Calidad según la conexión del usuario

## 🚀 Inicio Rápido

### Prerrequisitos

- Go 1.21 o superior
- PostgreSQL
- Biblioteca musical en formato MP3, FLAC, etc.

### Instalación

1. **Clona el repositorio**:
```bash
git clone <repository-url>
cd castafiore-backend
```

2. **Instala las dependencias**:
```bash
go mod download
```

3. **Configura las variables de entorno**:
```bash
export PORT=8080
export DATABASE_URL="postgres://user:password@localhost/castafiore?sslmode=disable"
export JWT_SECRET="your-secret-key-change-this-in-production"
export MUSIC_PATH="./music"
export MAX_CONCURRENT_STREAMS=3
export MAX_DOWNLOADS_PER_DAY=50
```

4. **Ejecuta el servidor**:
```bash
go run cmd/server/main.go
```

El servidor estará disponible en `http://localhost:8080`

## 🌐 Acceso Externo

Para permitir que otros usuarios accedan a tu servidor Castafiore desde fuera de tu red local:

### Configuración Automática (Recomendado)

1. **Ejecuta el script de configuración**:
```powershell
# En Windows
.\setup-external-access.ps1

# En Linux/macOS
chmod +x setup-external-access.sh && ./setup-external-access.sh
```

### Configuración Manual

1. **Configura las variables de entorno para acceso externo**:
```bash
# Permite acceso desde cualquier IP
export HOST=0.0.0.0
export PORT=8080
```

2. **Crea un archivo .env** desde `.env.example`:
```bash
cp .env.example .env
```

3. **Edita el archivo .env** y asegúrate de que `HOST=0.0.0.0`

4. **Configura tu firewall**:
   - **Windows**: Permite el puerto 8080 en Windows Defender Firewall
   - **Linux**: `sudo ufw allow 8080`

5. **Configura port forwarding en tu router**:
   - Accede a la configuración de tu router (192.168.1.1 o 192.168.0.1)
   - Crea una regla de port forwarding:
     - Puerto externo: 8080
     - Puerto interno: 8080  
     - IP interna: [tu-ip-local]
     - Protocolo: TCP

### Tipos de Acceso

- **Red local**: `http://[tu-ip-local]:8080`
- **Internet**: `http://[tu-ip-publica]:8080` (requiere port forwarding)

### Seguridad para Acceso Público

- ✅ Cambia `JWT_SECRET` por un valor aleatorio seguro
- ✅ Usa contraseñas fuertes para todos los usuarios
- ✅ Considera usar HTTPS con un proxy reverso (nginx, Cloudflare)
- ✅ Configura un dominio dinámico para IP cambiante
- ✅ Revisa regularmente los logs de acceso

## 📋 Endpoints de la API

### Sistema
- `GET /rest/ping` - Test de conectividad
- `GET /rest/getLicense` - Información de licencia
- `GET /health` - Estado del servidor

### Autenticación
Todos los endpoints requieren autenticación usando parámetros Subsonic:
- `u`: Nombre de usuario
- `p`: Contraseña (o token)
- `s`: Salt (opcional)
- `t`: Token MD5 (opcional)

### Navegación
- `GET /rest/getMusicFolders` - Carpetas de música
- `GET /rest/getIndexes` - Índice de artistas
- `GET /rest/getMusicDirectory` - Contenido de directorios
- `GET /rest/getArtists` - Lista de artistas
- `GET /rest/getArtist` - Detalles de artista
- `GET /rest/getAlbum` - Detalles de álbum

### Streaming
- `GET /rest/stream` - Streaming de archivos
- `GET /rest/download` - Descarga de archivos
- `GET /rest/getCoverArt` - Imágenes de carátula

## 🗄️ Base de Datos

El servidor utiliza PostgreSQL con las siguientes tablas principales:

- **users**: Información de usuarios y planes
- **artists**: Catálogo de artistas
- **albums**: Información de álbumes
- **songs**: Metadata de canciones
- **user_sessions**: Sesiones activas para control de concurrencia
- **downloads**: Registro de descargas para límites diarios

## 🏗️ Estructura del Proyecto

```
castafiore-backend/
├── cmd/
│   └── server/           # Punto de entrada
├── internal/
│   ├── api/             # Rutas de la API
│   ├── auth/            # Autenticación y autorización
│   ├── config/          # Configuración
│   ├── database/        # Conexión y migraciones
│   └── subsonic/        # Implementación API Subsonic
├── pkg/                 # Código reutilizable
├── configs/             # Archivos de configuración
├── migrations/          # Migraciones de base de datos
└── scripts/             # Scripts de utilidad
```

## 🛠️ Desarrollo

### Comandos útiles

```bash
# Ejecutar en modo desarrollo
go run cmd/server/main.go

# Compilar
go build -o bin/castafiore cmd/server/main.go

# Ejecutar tests
go test ./...

# Formatear código
go fmt ./...

# Linter
golangci-lint run
```

### Variables de Entorno

| Variable | Descripción | Valor por defecto |
|----------|-------------|-------------------|
| `PORT` | Puerto del servidor | `8080` |
| `DATABASE_URL` | URL de conexión a PostgreSQL | `postgres://user:password@localhost/castafiore?sslmode=disable` |
| `JWT_SECRET` | Clave secreta para JWT | `your-secret-key-change-this-in-production` |
| `MUSIC_PATH` | Ruta a la biblioteca musical | `./music` |
| `MAX_CONCURRENT_STREAMS` | Streams simultáneos por usuario | `3` |
| `MAX_DOWNLOADS_PER_DAY` | Descargas diarias por usuario | `50` |

## 🔧 Configuración

### Base de Datos

Configura PostgreSQL y crea la base de datos:

```sql
CREATE DATABASE castafiore;
CREATE USER castafiore_user WITH ENCRYPTED PASSWORD 'your_password';
GRANT ALL PRIVILEGES ON DATABASE castafiore TO castafiore_user;
```

### Biblioteca Musical

Organiza tu música en la siguiente estructura:
```
music/
├── Artist 1/
│   ├── Album 1/
│   │   ├── 01 - Song 1.mp3
│   │   ├── 02 - Song 2.mp3
│   │   └── cover.jpg
│   └── Album 2/
└── Artist 2/
```

## 🔒 Seguridad

- Las contraseñas se almacenan usando bcrypt
- Soporte para autenticación con salt/token estilo Subsonic
- JWT para sesiones de API
- Validación de entrada en todos los endpoints

## 📖 API Subsonic

Compatible con Subsonic API versión 1.16.1. Documentación completa disponible en:
- [Subsonic API Documentation](http://www.subsonic.org/pages/api.jsp)

## 🤝 Contribución

1. Fork el proyecto
2. Crea una rama para tu feature (`git checkout -b feature/AmazingFeature`)
3. Commit tus cambios (`git commit -m 'Add some AmazingFeature'`)
4. Push a la rama (`git push origin feature/AmazingFeature`)
5. Abre un Pull Request

## 📄 Licencia

Este proyecto está bajo la Licencia MIT. Ver el archivo `LICENSE` para más detalles.

## 🎯 Roadmap

- [ ] Scanner automático de biblioteca musical
- [ ] Interfaz web de administración
- [ ] Soporte para múltiples formatos de audio
- [ ] Cache inteligente para mejor rendimiento
- [ ] Análisis de uso y estadísticas
- [ ] Integración con servicios de metadata
- [ ] Soporte para playlists inteligentes
- [ ] API GraphQL

## 🔧 Solución de Problemas Comunes

### Error de autenticación de base de datos

Si ves el error: `pq: password authentication failed for user "user"`

**Solución:**
1. **Verifica que Docker Compose esté ejecutándose:**
   ```bash
   docker-compose ps
   ```

2. **Asegúrate de que las credenciales coincidan:**
   - El archivo `.env` debe usar: `postgres://castafiore_user:castafiore_password@localhost/castafiore?sslmode=disable`
   - O usa el script PowerShell que carga automáticamente el .env: `.\start.ps1`

3. **Reinicia los servicios si es necesario:**
   ```bash
   docker-compose down
   docker-compose up -d
   ```

### Variables de entorno no detectadas

**Opción 1: Script PowerShell (Recomendado)**
```bash
powershell -ExecutionPolicy Bypass -File start.ps1
```

**Opción 2: Cargar manualmente**
```bash
$env:DATABASE_URL="postgres://castafiore_user:castafiore_password@localhost/castafiore?sslmode=disable"
go run cmd/server/main.go
```

### Puerto ya en uso

Si ves: `bind: Only one usage of each socket address`
- Cambia el puerto en `.env`: `PORT=8097`
- O detén el proceso anterior: `Ctrl+C` en la terminal
