# Solución para Error JSON en Base de Datos

## Problema Identificado

El error `ERROR: invalid input syntax for type json (SQLSTATE 22P02)` se produce cuando GORM intenta insertar datos que no son JSON válido en campos definidos como `JSONB` en PostgreSQL.

## Causa Raíz

1. **Campo `metadata` en `execution_steps`**: Definido como `JSONB` en la migración SQL pero manejado como `string` en el modelo Go sin validación JSON.
2. **Arrays JSON nulos**: Los campos `approved_test_ids` y `failed_test_ids` pueden ser `nil`, causando problemas de serialización.
3. **Falta de validación**: No había validación para asegurar que los datos sean JSON válido antes de la persistencia.

## Solución Implementada

### 1. Función de Validación JSON Robusta

```go
func (s *ExecutionService) ensureValidJSON(payload interface{}) string {
    if payload == nil {
        return "{}" // fallback seguro
    }

    // Si es string, validar que sea JSON válido
    if str, ok := payload.(string); ok {
        if str == "" {
            return "{}"
        }
        
        var js interface{}
        if err := json.Unmarshal([]byte(str), &js); err != nil {
            // Si no es JSON válido, envolverlo como mensaje
            validPayload := map[string]interface{}{
                "message": str,
                "error":   "invalid_json_format",
            }
            if jsonBytes, err := json.Marshal(validPayload); err == nil {
                return string(jsonBytes)
            }
            return "{}" // fallback último
        }
        return str // JSON válido
    }

    // Para cualquier otro tipo, serializar a JSON
    jsonBytes, err := json.Marshal(payload)
    if err != nil {
        // Crear fallback seguro con información del error
        fallbackPayload := map[string]interface{}{
            "error":           "marshal_failed",
            "original_type":   fmt.Sprintf("%T", payload),
            "error_message":   err.Error(),
        }
        if jsonBytes, err := json.Marshal(fallbackPayload); err == nil {
            return string(jsonBytes)
        }
        return "{}" // fallback último
    }

    return string(jsonBytes)
}
```

### 2. Inicialización Segura de Arrays JSON

```go
// En CreateExecution
execution := &models.Execution{
    // ... otros campos ...
    ApprovedTestIDs: []string{}, // Inicializar array vacío
    FailedTestIDs:   []string{}, // Inicializar array vacío
}

// En CompleteExecution
if approvedTestIDs == nil {
    execution.ApprovedTestIDs = []string{}
} else {
    execution.ApprovedTestIDs = approvedTestIDs
}

if failedTestIDs == nil {
    execution.FailedTestIDs = []string{}
} else {
    execution.FailedTestIDs = failedTestIDs
}
```

### 3. Validación en Operaciones de Steps

```go
// En CompleteExecutionStep
step.Metadata = s.ensureValidJSON(metadata)

// En AddExecutionStep
Metadata: s.ensureValidJSON(nil), // Inicializar con JSON vacío

// En StartExecution
Metadata: s.ensureValidJSON(map[string]interface{}{
    "step_type": "initialization",
    "started_at": time.Now().Format(time.RFC3339),
}),
```

## Beneficios de la Solución

1. **Robustez**: Nunca falla al insertar datos en campos JSON/JSONB
2. **Fallback Seguro**: Siempre retorna JSON válido, incluso en casos de error
3. **Información de Debugging**: En caso de errores, conserva información útil
4. **Compatibilidad**: Funciona con strings, structs, maps, arrays, etc.
5. **Prevención**: Evita completamente el error `SQLSTATE 22P02`

## Archivos Modificados

- `internal/services/execution_service.go`: Añadida función `ensureValidJSON` y actualización de métodos
- Se asegura validación JSON antes de toda persistencia
- Inicialización segura de arrays JSON

## Verificación

La solución ha sido probada y el proyecto compila exitosamente sin errores.