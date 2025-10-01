# 🐳 Docker Module - Setup Instructions

## ✅ Implementación Completada

Se ha creado un módulo completo de ejecución de Docker que:

1. ✅ Recibe el código generado (template C++ con doctest)
2. ✅ Lo ejecuta en un contenedor Docker aislado
3. ✅ Captura los resultados de los tests
4. ✅ Actualiza la base de datos con los resultados
5. ✅ Aplica límites de seguridad (CPU, memoria, timeout, sin red)

## 🚀 Próximos Pasos

### 1. Iniciar Docker Desktop

**IMPORTANTE:** Asegúrate de que Docker Desktop está corriendo en tu Mac.

```bash
# Verificar que Docker está funcionando
docker version
```

Si Docker no está corriendo, verás un error. Inicia Docker Desktop desde Aplicaciones.

### 2. Construir la Imagen Docker de C++

Antes de que el servidor pueda ejecutar código, necesitas construir la imagen Docker:

```bash
# Opción 1: Desde el directorio raíz
docker build -t coderunner-cpp:latest -f docker/cpp/Dockerfile docker/cpp/

# Opción 2: Usando el script (más fácil)
./docker/build-images.sh
```

**Salida esperada:**
```
🔨 Building C++ Docker image...
[+] Building 45.2s (10/10) FINISHED
 => [1/5] FROM docker.io/library/gcc:13.2
 => [2/5] RUN apt-get update && apt-get install...
 ...
✅ C++ Docker image built successfully!
```

### 3. Verificar la Imagen

```bash
docker images coderunner-cpp:latest
```

Deberías ver algo como:
```
REPOSITORY          TAG       IMAGE ID       CREATED         SIZE
coderunner-cpp      latest    abc123def456   2 minutes ago   1.5GB
```

### 4. Iniciar el Servidor

```bash
go run ./cmd/server/main.go
```

**Logs esperados:**
```
2025/10/01 14:47:00 ✅ Database connected successfully
2025/10/01 14:47:00 ✅ Database migration completed successfully
2025/10/01 14:47:00 🚀 Starting gRPC server on port 9084
```

**Si Docker NO está disponible, verás:**
```
⚠️  Warning: Failed to create Docker executor: ...
⚠️  Docker execution will not be available. Make sure Docker is running.
```

### 5. Probar desde Spring Boot

Envía una petición gRPC desde tu cliente Spring Boot. El servidor ahora:

1. ✅ Recibirá el código
2. ✅ Creará el template C++ con doctest
3. ✅ **Ejecutará el código en Docker** ⬅️ NUEVO
4. ✅ **Capturará los resultados de los tests** ⬅️ NUEVO
5. ✅ Guardará todo en la base de datos
6. ✅ Retornará los resultados

## 📋 Logs de Ejecución Exitosa

Cuando ejecutes código, verás logs como estos:

```
🚀 ===== RECEIVED EXECUTION REQUEST =====
  📋 Challenge ID: 63f9587e-01f8-4c2f-adc4-0bab150dde34
  🔢 Code Version ID: cc68bb60-1081-4ed6-bd9d-f6caf1ae149e
  👤 Student ID: 660e8400-e29b-41d4-a716-446655440001
📝 Creating execution record...
✅ Execution record created with ID: 0a92b41d-3639-40cf-8e04-4906d3c03a88
🔧 Generating C++ execution template...
✅ Template generated and saved to database
  📄 Template ID: d4c6559a-9de1-4138-8b6c-84cebb495532
  📏 Template size: 548 bytes

🐳 Executing code in Docker container...          ⬅️ NUEVO
  📁 Temp directory created: /tmp/coderunner-xxx  ⬅️ NUEVO
  💾 Source code saved: solution.cpp              ⬅️ NUEVO
  ✅ Image coderunner-cpp:latest found            ⬅️ NUEVO
  🔧 Container configured: Memory=256MB, CPU=0.5   ⬅️ NUEVO
  ✅ Container created: abcd1234                   ⬅️ NUEVO
  🚀 Container started                             ⬅️ NUEVO
  ✅ Container finished with exit code: 0         ⬅️ NUEVO
  📊 Execution completed in 123ms                  ⬅️ NUEVO
  🧪 Test results: 3/3 passed                     ⬅️ NUEVO
  🧹 Cleaning up container                        ⬅️ NUEVO

✅ Docker execution completed                      ⬅️ NUEVO
  ⏱️  Execution time: 123 ms                      ⬅️ NUEVO
  📊 Exit code: 0                                  ⬅️ NUEVO
  🧪 Tests: 3/3 passed                            ⬅️ NUEVO

✅ ===== EXECUTION COMPLETED =====
```

## 🔍 Verificar en la Base de Datos

Puedes consultar los resultados en PostgreSQL:

```sql
-- Ver ejecuciones
SELECT 
    id,
    challenge_id,
    student_id,
    status,
    success,
    passed_tests,
    total_tests,
    execution_time_ms,
    created_at
FROM executions
ORDER BY created_at DESC
LIMIT 5;

-- Ver código generado
SELECT 
    id,
    execution_id,
    language,
    test_cases_count,
    generation_time_ms,
    code_size_bytes
FROM generated_test_code
ORDER BY created_at DESC
LIMIT 5;
```

## 🛡️ Configuraciones de Seguridad Activas

El contenedor Docker tiene las siguientes restricciones:

- ✅ **Memoria máxima:** 256 MB
- ✅ **CPU máximo:** 50% de un core
- ✅ **Timeout:** 30 segundos
- ✅ **Red:** Deshabilitada (sin acceso a internet)
- ✅ **Sistema de archivos:** Solo lectura
- ✅ **Usuario:** No-root (coderunner, UID 1000)
- ✅ **Capabilities:** Todas eliminadas
- ✅ **Contenedor:** Se destruye después de cada ejecución

## 📚 Documentación Adicional

- **Detalles técnicos:** Ver `DOCKER_IMPLEMENTATION.md`
- **Guía de uso:** Ver `docker/README.md`
- **Dockerfile:** Ver `docker/cpp/Dockerfile`
- **Código fuente:** Ver `internal/docker/executor.go`

## ⚠️ Troubleshooting

### Error: "Cannot connect to Docker daemon"

**Problema:** Docker Desktop no está corriendo.

**Solución:**
1. Abre Docker Desktop desde Aplicaciones
2. Espera a que aparezca el ícono de Docker en la barra de menú
3. Verifica con `docker version`

### Error: "Image not found: coderunner-cpp:latest"

**Problema:** La imagen Docker no ha sido construida.

**Solución:**
```bash
cd docker
docker build -t coderunner-cpp:latest -f cpp/Dockerfile .
```

### Warning: "Docker executor not available"

**Problema:** No se pudo conectar a Docker.

**Efecto:** El servidor funcionará pero solo generará templates, no ejecutará código.

**Solución:** Inicia Docker Desktop y reinicia el servidor.

## 🎯 Flujo Completo

```
Spring Boot Client
       ↓
   gRPC Request (código + tests)
       ↓
   Go Server recibe request
       ↓
   Crea registro en DB (executions)
       ↓
   Genera template C++ con doctest
       ↓
   Guarda template en DB (generated_test_code)
       ↓
   🆕 Crea contenedor Docker
       ↓
   🆕 Monta código como volumen
       ↓
   🆕 Ejecuta: g++ + ./solution
       ↓
   🆕 Captura stdout/stderr
       ↓
   🆕 Parsea resultados de doctest
       ↓
   🆕 Destruye contenedor
       ↓
   Actualiza DB con resultados
       ↓
   gRPC Response (resultados + tests pasados)
       ↓
   Spring Boot Client
```

## ✅ Checklist Final

Antes de probar con Spring Boot, verifica:

- [ ] Docker Desktop está corriendo (`docker version` funciona)
- [ ] Imagen construida (`docker images coderunner-cpp:latest`)
- [ ] PostgreSQL corriendo y migrado
- [ ] Servidor Go iniciado sin warnings de Docker
- [ ] Puerto 9084 disponible

¡Listo para probar! 🚀
