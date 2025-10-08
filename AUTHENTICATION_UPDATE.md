# Actualización de Autenticación - Ping y GetLicense

## Problema Resuelto

**Síntoma:** Los clientes Subsonic podían "conectarse" con credenciales inválidas porque el endpoint `/rest/ping.view` retornaba HTTP 200 sin validar las credenciales.

**Ejemplo:**
```
Usuario: sdf (no existe)
Contraseña: cualquiera
Resultado: HTTP 200 OK - El cliente pensaba que la conexión era exitosa
```

## Solución Implementada

Se agregó autenticación obligatoria a los endpoints:
- `/rest/ping`
- `/rest/ping.view`
- `/rest/getLicense`
- `/rest/getLicense.view`

### Cambios en el Código

**Archivo:** `internal/api/routes.go`

**Antes:**
```go
rest.GET("/ping", subsonicService.Ping)
rest.GET("/ping.view", subsonicService.Ping)
rest.GET("/getLicense", subsonicService.GetLicense)
rest.GET("/getLicense.view", subsonicService.GetLicense)
```

**Después:**
```go
rest.GET("/ping", subsonicService.AuthMiddleware(), subsonicService.Ping)
rest.GET("/ping.view", subsonicService.AuthMiddleware(), subsonicService.Ping)
rest.GET("/getLicense", subsonicService.AuthMiddleware(), subsonicService.GetLicense)
rest.GET("/getLicense.view", subsonicService.AuthMiddleware(), subsonicService.GetLicense)
```

## Comportamiento Actual

### ✅ Con Credenciales Válidas
```bash
GET /rest/ping.view?u=antonio&t={token_válido}&s={salt}&v=1.16.1&c=Client&f=json

Respuesta:
{
  "subsonic-response": {
    "status": "ok",
    "version": "1.16.1",
    "type": "castafiore"
  }
}
```

### ❌ Con Credenciales Inválidas
```bash
GET /rest/ping.view?u=usuario_invalido&t={token}&s={salt}&v=1.16.1&c=Client&f=json

Respuesta:
{
  "subsonic-response": {
    "status": "failed",
    "version": "1.16.1",
    "type": "castafiore",
    "error": {
      "code": 40,
      "message": "Wrong username or password"
    }
  }
}
```

## Nota sobre la Especificación Subsonic

La especificación oficial de Subsonic API indica que los endpoints `ping` y `getLicense` **NO deben requerir autenticación**. Sin embargo, hemos decidido desviarnos de esta especificación por las siguientes razones:

1. **Seguridad mejorada:** Todos los endpoints validan credenciales
2. **Mejor experiencia de usuario:** Los clientes rechazan inmediatamente credenciales inválidas
3. **Prevención de confusión:** Evita que usuarios piensen que están conectados cuando no lo están
4. **Compatibilidad:** La mayoría de clientes Subsonic modernos envían credenciales en ping de todas formas

## Pruebas Realizadas

✅ **Test 1:** Ping con credenciales inválidas → RECHAZADO (error 40)  
✅ **Test 2:** Ping con credenciales válidas → ACEPTADO (status ok)  
✅ **Test 3:** GetLicense con credenciales inválidas → RECHAZADO (error 40)  
✅ **Test 4:** GetLicense con credenciales válidas → ACEPTADO (status ok)  
✅ **Test 5:** GetMusicFolders con credenciales válidas → ACEPTADO (retorna carpetas)  
✅ **Test 6:** GetMusicFolders con credenciales inválidas → RECHAZADO (error 40)  

## Cómo Aplicar

1. **Detener el servidor actual:**
   ```powershell
   Get-Process -Name "castafiore*" | Stop-Process -Force
   ```

2. **Recompilar:**
   ```powershell
   cd c:\repositorios\castafiore-backend
   go build -o bin/castafiore.exe cmd/server/main.go
   ```

3. **Iniciar el servidor:**
   ```powershell
   .\bin\castafiore.exe
   ```

## Credenciales de Prueba

| Usuario | Contraseña | Rol |
|---------|------------|-----|
| admin | admin123 | Administrador |
| antonio | 150291 | Usuario |
| fredyaran | Aleida2001+ | Usuario |

## Verificación

Prueba la conexión desde tu cliente Subsonic:
- **URL del servidor:** `http://localhost:8080` (o tu IP)
- **Usuario:** antonio
- **Contraseña:** 150291

El cliente ahora debe:
- ✅ Aceptar credenciales válidas
- ❌ Rechazar credenciales inválidas inmediatamente
- ✅ Mostrar mensaje de error claro cuando las credenciales son incorrectas

## Fecha de Implementación

7 de octubre de 2025

## Estado

✅ **IMPLEMENTADO Y PROBADO**