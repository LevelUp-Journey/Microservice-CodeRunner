# 📁 Almacenamiento Local de Códigos Compilados

## ✅ Cambio Implementado

Los archivos de ejecución de Docker ahora se almacenan de manera permanente en el directorio local `compiled_test_codes/` en lugar de usar directorios temporales que se eliminan después de cada ejecución.

## 🔄 Antes vs Después

### Antes
```go
// Usaba directorio temporal
tempDir, err := os.MkdirTemp("", fmt.Sprintf("coderunner-%s", config.ExecutionID.String()))
defer os.RemoveAll(tempDir) // Se eliminaba después de ejecutar

// Ruta: /var/folders/zm/1rn3w8nj2xv4ncdrkm306yn00000gn/T/coderunner-{uuid}/solution.cpp
```

### Después
```go
// Usa directorio local permanente
baseDir := "compiled_test_codes"
executionDir := filepath.Join(baseDir, config.ExecutionID.String())
os.MkdirAll(executionDir, 0755)

// Ruta: compiled_test_codes/{execution-id}/solution.cpp
// NO se elimina después de ejecutar
```

## 📂 Estructura de Archivos

```
Microservice-CodeRunner/
├── compiled_test_codes/          # ⬅️ NUEVO
│   ├── README.md                 # Documentación
│   ├── {execution-id-1}/
│   │   └── solution.cpp          # Código generado
│   ├── {execution-id-2}/
│   │   └── solution.cpp
│   └── {execution-id-3}/
│       └── solution.cpp
├── cmd/
├── internal/
└── ...
```

## 🎯 Beneficios

### 1. **Auditoría**
```bash
# Ver el código generado de cualquier ejecución
cat compiled_test_codes/c162f359-8237-40cf-a3c2-dd209c1cd23c/solution.cpp
```

### 2. **Debugging**
```bash
# Compilar y ejecutar manualmente
cd compiled_test_codes/c162f359-8237-40cf-a3c2-dd209c1cd23c
g++ -std=c++17 solution.cpp -o solution
./solution
```

### 3. **Análisis Post-Mortem**
```bash
# Revisar códigos que fallaron
grep -r "ERROR" compiled_test_codes/*/solution.cpp
```

### 4. **Estadísticas**
```bash
# Ver cuántas ejecuciones se han hecho
ls -1 compiled_test_codes | wc -l

# Ver tamaño total
du -sh compiled_test_codes/
```

## 📊 Integración con Base de Datos

El `ExecutionID` en el nombre del directorio coincide con el ID en la base de datos:

```sql
-- Consultar ejecución y su archivo
SELECT 
    e.id,
    e.challenge_id,
    e.student_id,
    e.status,
    CONCAT('compiled_test_codes/', e.id::text, '/solution.cpp') as file_path,
    e.execution_time_ms,
    e.created_at
FROM executions e
WHERE e.id = 'c162f359-8237-40cf-a3c2-dd209c1cd23c';
```

## 🔧 Configuración de Docker

### Cambios Realizados

1. **Montaje de Volumen**: Cambiado de read-only (`:ro`) a read-write para permitir compilación
2. **ReadonlyRootfs**: Deshabilitado para permitir archivos temporales del compilador

```go
// Antes
Binds: []string{
    fmt.Sprintf("%s:%s:ro", tempDir, config.WorkDir),
},
ReadonlyRootfs: e.dockerConfig.ReadOnlyRootFS,

// Ahora
Binds: []string{
    fmt.Sprintf("%s:%s", executionDir, config.WorkDir), // Sin :ro
},
// ReadonlyRootfs comentado
```

### Seguridad Mantenida

Aunque se permite escritura en el directorio montado:
- ✅ Sin acceso a red (`NetworkMode: "none"`)
- ✅ Sin capabilities (`CapDrop: ["ALL"]`)
- ✅ No new privileges (`SecurityOpt: ["no-new-privileges"]`)
- ✅ Límites de memoria y CPU
- ✅ Timeout de 30 segundos
- ✅ Contenedor se destruye después de ejecutar

## 🔍 Verificación

### 1. Ejecutar una Petición

Envía una petición desde Spring Boot al microservicio Go.

### 2. Verificar Archivo Creado

```bash
# Listar ejecuciones
ls -la compiled_test_codes/

# Ver el último archivo creado
ls -lt compiled_test_codes/ | head -2

# Ver contenido
cat compiled_test_codes/{execution-id}/solution.cpp
```

### 3. Logs Esperados

```
2025/10/01 15:10:45 🐳 Starting Docker execution for ExecutionID: c162f359-8237-40cf-a3c2-dd209c1cd23c
2025/10/01 15:10:45   📁 Execution directory created: compiled_test_codes/c162f359-8237-40cf-a3c2-dd209c1cd23c
2025/10/01 15:10:45   💾 Source code saved: compiled_test_codes/c162f359-8237-40cf-a3c2-dd209c1cd23c/solution.cpp (548 bytes)
```

## 🧹 Mantenimiento

### Limpieza Manual

```bash
# Eliminar archivos más antiguos de 7 días
find compiled_test_codes -type d -mtime +7 -exec rm -rf {} +

# Eliminar todo excepto README
find compiled_test_codes -mindepth 1 -maxdepth 1 -type d -exec rm -rf {} +

# Ver archivos más antiguos
find compiled_test_codes -type f -mtime +7 | head -10
```

### Script de Limpieza (Opcional)

```bash
#!/bin/bash
# cleanup_old_executions.sh

DAYS=7
BASE_DIR="compiled_test_codes"

echo "🧹 Limpiando archivos más antiguos de $DAYS días..."

# Contar archivos a eliminar
COUNT=$(find "$BASE_DIR" -mindepth 1 -maxdepth 1 -type d -mtime +$DAYS | wc -l)

if [ "$COUNT" -eq 0 ]; then
    echo "✅ No hay archivos antiguos para limpiar"
    exit 0
fi

echo "📋 Se eliminarán $COUNT directorios"

# Eliminar
find "$BASE_DIR" -mindepth 1 -maxdepth 1 -type d -mtime +$DAYS -exec rm -rf {} +

echo "✅ Limpieza completada"
```

### Cron Job (Opcional)

```cron
# Limpiar cada semana
0 2 * * 0 /path/to/cleanup_old_executions.sh
```

## ⚠️ Consideraciones

### Espacio en Disco

```bash
# Monitorear tamaño
du -sh compiled_test_codes/

# Si crece mucho, configurar limpieza automática
```

### Git

El directorio está en `.gitignore`:
```gitignore
# Códigos compilados de tests
/compiled_test_codes/
```

Solo el README.md se incluye en el repo para documentación.

### Backups

Si necesitas respaldar estos archivos:
```bash
# Crear backup
tar -czf compiled_test_codes_backup_$(date +%Y%m%d).tar.gz compiled_test_codes/

# Restaurar
tar -xzf compiled_test_codes_backup_20251001.tar.gz
```

## 📝 Archivos Modificados

1. `internal/docker/executor.go`
   - Cambiado de `os.MkdirTemp()` a `os.MkdirAll()`
   - Removido `defer os.RemoveAll()`
   - Actualizado path a `compiled_test_codes/{execution-id}/`
   - Deshabilitado ReadonlyRootfs
   - Removido `:ro` del bind mount

2. `.gitignore`
   - Agregado `/compiled_test_codes/` para excluir del repo

3. `compiled_test_codes/README.md`
   - Creado para documentar la carpeta

## ✅ Testing

```bash
# 1. Iniciar servidor
go run ./cmd/server/main.go

# 2. Enviar petición desde Spring Boot

# 3. Verificar archivo creado
ls compiled_test_codes/

# 4. Ver contenido
cat compiled_test_codes/{execution-id}/solution.cpp

# 5. Compilar manualmente
cd compiled_test_codes/{execution-id}
g++ -std=c++17 solution.cpp -o solution
./solution
```

¡Implementación completada! 🎉
