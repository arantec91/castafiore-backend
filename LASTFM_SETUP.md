# Configuración de Last.fm para Canciones Similares

## Obtener API Key de Last.fm

1. Visita [https://www.last.fm/api/account/create](https://www.last.fm/api/account/create)
2. Crea una cuenta de desarrollador si no tienes una
3. Registra tu aplicación con la información requerida:
   - Application name: `Castafiore Music Server`
   - Application description: `Personal music streaming server with Subsonic API compatibility`
   - Application homepage URL: `http://localhost:8080` (o tu dominio)
   - Callback URL: Puedes dejarlo vacío para esta implementación

4. Una vez creada la aplicación, obtendrás:
   - **API key**: Esta es la que necesitas
   - **Shared secret**: No necesaria para esta implementación (solo para operaciones que requieren autenticación)

## Configuración

### Opción 1: Variable de Entorno (Recomendado)
```bash
# Windows PowerShell
$env:LASTFM_API_KEY = "tu_api_key_aqui"

# Windows CMD
set LASTFM_API_KEY=tu_api_key_aqui

# Linux/Mac
export LASTFM_API_KEY="tu_api_key_aqui"
```

### Opción 2: Archivo .env
Crea un archivo `.env` en la raíz del proyecto:
```
LASTFM_API_KEY=tu_api_key_aqui
```

### Opción 3: Hardcodeado (Solo para desarrollo)
Edita el archivo `internal/lastfm/service.go` y reemplaza:
```go
APIKey = "YOUR_LASTFM_API_KEY"
```
con:
```go
APIKey = "tu_api_key_aqui"
```

## Uso

Una vez configurado, el endpoint `getSimilarSongs2` funcionará de la siguiente manera:

1. **Primario**: Busca canciones similares usando Last.fm API
2. **Fallback**: Si Last.fm no tiene datos o falla, usa estrategias locales:
   - Canciones del mismo artista
   - Canciones del mismo álbum
   - Canciones del mismo género

## Pruebas

### Scripts de Prueba Incluidos

**PowerShell (Windows):**
```powershell
# Usar valores por defecto
.\test-similar-songs.ps1

# Especificar parámetros
.\test-similar-songs.ps1 -SongId 123 -Count 20
```

**Bash (Linux/Mac):**
```bash
# Usar valores por defecto
./test-similar-songs.sh

# Especificar parámetros
./test-similar-songs.sh 123 20
```

### Prueba Manual con curl

```bash
# Obtener canciones similares a la canción con ID 6201
curl "http://localhost:8080/rest/getSimilarSongs2.view?u=demo&p=demo&v=1.16.1&c=Castafiore&id=6201&count=15&f=json"
```

## Parámetros

- `id` (requerido): ID de la canción de referencia
- `count` (opcional): Número de canciones similares a devolver (default: 50, máximo: 500)

## Límites de Last.fm

- **Rate Limit**: 5 requests por segundo por IP
- **Límite diario**: Generalmente muy alto para uso personal
- **Datos**: Depende de la popularidad de la música en Last.fm

## Troubleshooting

### Error "Last.fm API error 6: No such track found"
- La canción no existe en la base de datos de Last.fm
- El sistema usará automáticamente estrategias de fallback

### Error "Last.fm API returned status 403"
- API key inválida o no configurada
- Verifica que la API key esté correctamente configurada

### Error "context deadline exceeded"
- Problemas de conectividad con Last.fm
- El sistema usará automáticamente estrategias de fallback

## Rendimiento

Para optimizar el rendimiento, considera:

1. **Cache**: Implementar cache de respuestas de Last.fm
2. **Batch processing**: Agrupar múltiples requests
3. **Background updates**: Actualizar similitudes en background
