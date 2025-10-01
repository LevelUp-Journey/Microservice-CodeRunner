# Docker Execution Module

Este módulo maneja la ejecución segura de código C++ en contenedores Docker aislados.

## 🏗️ Arquitectura

### Componentes

1. **DockerExecutor** (`internal/docker/executor.go`)
   - Gestiona la ejecución de código en contenedores Docker
   - Configura límites de recursos (CPU, memoria, timeouts)
   - Captura stdout/stderr y parsea resultados de tests

2. **Types** (`internal/docker/types.go`)
   - `ExecutionConfig`: Configuración de ejecución (límites, timeouts, imagen)
   - `ExecutionResult`: Resultados de ejecución (output, tests, métricas)
   - `DockerConfig`: Configuración global de Docker

3. **Dockerfile** (`docker/cpp/Dockerfile`)
   - Imagen base con GCC 13.2
   - Incluye doctest framework
   - Usuario no-root para seguridad

## 🚀 Setup

### 1. Iniciar Docker Desktop

Asegúrate de que Docker Desktop está corriendo:

```bash
docker version
```

### 2. Construir la imagen de C++

```bash
cd docker
docker build -t coderunner-cpp:latest -f cpp/Dockerfile .
```

O usa el script de construcción:

```bash
./docker/build-images.sh
```

### 3. Verificar la imagen

```bash
docker images coderunner-cpp:latest
```

## 🔧 Configuración

### Límites de Recursos (por defecto)

- **Memoria**: 256 MB
- **CPU**: 50% de un core (0.5)
- **Timeout**: 30 segundos

### Configuración de Seguridad

- **NetworkMode**: `none` (sin acceso a red)
- **ReadOnlyRootFS**: `true` (sistema de archivos de solo lectura)
- **DropCapabilities**: `ALL` (sin capacidades especiales)
- **SecurityOpt**: `no-new-privileges`

## 📋 Uso

### Ejecución Automática

El servidor ejecuta automáticamente el código en Docker después de generar el template:

1. Se crea el registro de ejecución en la base de datos
2. Se genera el template C++ con doctest
3. Se ejecuta en un contenedor Docker aislado
4. Se parsean los resultados y se actualizan en la base de datos

### Ejecución Manual

```go
import "code-runner/internal/docker"

// Crear executor
executor, err := docker.NewDockerExecutor()
if err != nil {
    log.Fatal(err)
}
defer executor.Close()

// Configurar ejecución
config := docker.DefaultExecutionConfig(executionID, sourceCode)

// Ejecutar
result, err := executor.Execute(context.Background(), config)
if err != nil {
    log.Fatal(err)
}

// Procesar resultados
fmt.Printf("Success: %v\n", result.Success)
fmt.Printf("Tests passed: %d/%d\n", result.PassedTests, result.TotalTests)
```

## 🧪 Formato de Salida de Tests

El sistema parsea la salida de doctest automáticamente:

```
[doctest] doctest version is "2.4.11"
[doctest] run with "--help" for options
===============================================================================
[doctest] test cases:  1 |  1 passed | 0 failed | 0 skipped
[doctest] assertions:  3 |  3 passed | 0 failed |
```

## 🔍 Troubleshooting

### Docker no está disponible

Si ves el warning:
```
⚠️  Warning: Failed to create Docker executor
⚠️  Docker execution will not be available
```

**Solución**: Inicia Docker Desktop

### Imagen no encontrada

Si ves el error:
```
🔨 Image coderunner-cpp:latest not found, building...
```

**Solución**: Construye la imagen manualmente:
```bash
cd docker
docker build -t coderunner-cpp:latest -f cpp/Dockerfile .
```

### Timeout de ejecución

Si el código tarda más de 30 segundos:
- Ajusta `TimeoutSeconds` en `ExecutionConfig`
- Verifica que el código no tiene loops infinitos

### Error de permisos

Si hay errores de permisos en el contenedor:
- El contenedor usa usuario `coderunner` (UID 1000)
- Los archivos se montan como read-only

## 📊 Métricas de Ejecución

Cada ejecución registra:

- ⏱️ **Tiempo de ejecución** (ms)
- 💾 **Uso de memoria** (MB)
- 🧪 **Tests pasados/totales**
- 📤 **Stdout/Stderr**
- ❌ **Errores de compilación/runtime**
- ⏰ **Timeouts**

## 🛡️ Seguridad

### Aislamiento

- Contenedores efímeros (se destruyen después de cada ejecución)
- Sin acceso a red
- Sistema de archivos read-only
- Sin privilegios especiales

### Límites

- Memoria limitada para prevenir OOM
- CPU limitada para prevenir uso excesivo
- Timeout para prevenir ejecuciones infinitas

## 🔮 Próximos Pasos

- [ ] Soporte para Python y Java
- [ ] Métricas avanzadas (CPU usage, I/O)
- [ ] Cache de imágenes Docker
- [ ] Pools de contenedores pre-calentados
- [ ] Logs estructurados para análisis
