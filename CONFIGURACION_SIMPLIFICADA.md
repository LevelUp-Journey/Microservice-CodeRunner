# 🎯 Configuración Simplificada - CodeRunner Microservice

## 📋 Resumen de Cambios

Se ha simplificado significativamente la configuración del microservicio. **Solo necesitas configurar 8 variables esenciales** en lugar de 31.

---

## ✅ Variables REQUERIDAS (8 variables)

Estas son las **ÚNICAS** variables que necesitas configurar en tu archivo `.env`:

```bash
# ============================================================================
# CONFIGURACIÓN MÍNIMA REQUERIDA
# ============================================================================

# ----------------------------------------------------------------------------
# APLICACIÓN (Opcional - tiene valores por defecto)
# ----------------------------------------------------------------------------
APP_NAME=microservice-code-runner
API_VERSION=v1
PORT=8084
GRPC_PORT=9084

# ----------------------------------------------------------------------------
# BASE DE DATOS (4 variables requeridas)
# ----------------------------------------------------------------------------
DB_HOST=localhost                    # O Azure: *.postgres.database.azure.com
DB_PORT=5432
DB_USER=postgres                     # O Azure: admin@servername
DB_PASSWORD=postgres                 # Tu password seguro
DB_NAME=code_runner_db

# ----------------------------------------------------------------------------
# KAFKA / AZURE EVENT HUB (2 variables requeridas)
# ----------------------------------------------------------------------------
KAFKA_BOOTSTRAP_SERVERS=tu-namespace.servicebus.windows.net:9093
KAFKA_CONNECTION_STRING=Endpoint=sb://tu-namespace.servicebus.windows.net/;SharedAccessKeyName=RootManageSharedAccessKey;SharedAccessKey=tu-clave==

# ----------------------------------------------------------------------------
# SERVICE DISCOVERY (1 variable - auto-activa el servicio)
# ----------------------------------------------------------------------------
SERVICE_DISCOVERY_URL=http://eureka-server:8761/eureka
SERVICE_NAME=CODE-RUNNER-SERVICE
SERVICE_PUBLIC_IP=                   # Opcional: se auto-detecta
```

---

## ❌ Variables ELIMINADAS (hardcoded en código)

Estas variables YA NO son necesarias en `.env` porque están hardcoded en el código:

### Kafka/Event Hub
- ❌ `KAFKA_TOPIC` → Hardcoded: `challenge.completed` (puedes crear tópicos dinámicamente)
- ❌ `KAFKA_CONSUMER_GROUP` → Hardcoded: `code-runner-service`
- ❌ `KAFKA_SASL_MECHANISM` → Hardcoded: `PLAIN`
- ❌ `KAFKA_SECURITY_PROTOCOL` → Hardcoded: `SASL_SSL`
- ❌ `KAFKA_PRODUCER_TIMEOUT_MS` → Hardcoded: `30000`
- ❌ `KAFKA_CONSUMER_TIMEOUT_MS` → Hardcoded: `30000`
- ❌ `KAFKA_MAX_RETRIES` → Hardcoded: `3`

### Base de Datos
- ❌ `DB_SSLMODE` → Hardcoded: `disable` (o detecta automáticamente Azure)
- ❌ `DB_TIMEZONE` → Hardcoded: `UTC`
- ❌ `DB_MAX_OPEN_CONNS` → Hardcoded: `25`
- ❌ `DB_MAX_IDLE_CONNS` → Hardcoded: `10`
- ❌ `DB_CONN_MAX_LIFETIME` → Hardcoded: `3600` segundos

### Service Discovery
- ❌ `SERVICE_DISCOVERY_ENABLED` → Auto-detectado (true si `SERVICE_DISCOVERY_URL` existe)

### Logging
- ❌ `LOG_LEVEL` → Hardcoded: `info`
- ❌ `LOG_FORMAT` → Hardcoded: `json`

### pgAdmin (solo para desarrollo local)
- ❌ `PGADMIN_EMAIL` → Hardcoded en docker-compose
- ❌ `PGADMIN_PASSWORD` → Hardcoded en docker-compose
- ❌ `PGADMIN_PORT` → Hardcoded en docker-compose

---

## 🚀 Tópicos Dinámicos de Kafka

### ¿Qué cambió?

**ANTES:**
```bash
# Tenías que configurar el tópico en .env
KAFKA_TOPIC=challenge.completed
```

**AHORA:**
```go
// Creas y usas tópicos dinámicamente en código
kafkaClient.ProduceMessage(ctx, "challenge.completed", key, data)
kafkaClient.ProduceMessage(ctx, "student.registered", key, data)
kafkaClient.ProduceMessage(ctx, "custom.topic", key, data)
```

### Beneficios

✅ **Sin EntityPath en Connection String** - Ya no necesitas incluir `EntityPath=...`
✅ **Múltiples Tópicos** - Publica a cualquier tópico sin reconfigurar
✅ **Organización Flexible** - Crea estructura de tópicos según necesites
✅ **Multi-Tenant Ready** - Fácil separación por organización/cliente

### Ejemplos de Uso

```go
// Publicar a tópico por defecto
kafkaClient.PublishChallengeCompleted(ctx, event)

// Publicar a tópico específico
kafkaClient.PublishChallengeCompletedToTopic(ctx, "challenges.premium", event)

// Publicar a cualquier tópico personalizado
kafkaClient.ProduceMessage(ctx, "my.custom.topic", "key123", []byte("data"))

// Consumir de múltiples tópicos
topics := []string{"topic1", "topic2", "topic3"}
kafkaClient.InitConsumerForTopics(topics)
kafkaClient.ConsumeFromMultipleTopics(ctx, topics, handler)
```

**Ver documentación completa:** [docs/KAFKA_DYNAMIC_TOPICS.md](docs/KAFKA_DYNAMIC_TOPICS.md)

---

## 🔧 Service Discovery Auto-Activado

### ¿Qué cambió?

**ANTES:**
```bash
SERVICE_DISCOVERY_ENABLED=true
SERVICE_DISCOVERY_URL=http://eureka:8761/eureka
```

**AHORA:**
```bash
# Solo configura la URL, se activa automáticamente
SERVICE_DISCOVERY_URL=http://eureka:8761/eureka
```

### Lógica

```go
// En el código:
if config.ServiceDiscovery.URL != "" {
    config.ServiceDiscovery.Enabled = true
}
```

- ✅ Si `SERVICE_DISCOVERY_URL` tiene valor → Service Discovery **ACTIVADO**
- ✅ Si `SERVICE_DISCOVERY_URL` está vacío → Service Discovery **DESACTIVADO**

---

## 📝 Archivo .env Mínimo

Copia esto a tu archivo `.env` y solo edita los valores:

```bash
# Application (opcional)
APP_NAME=microservice-code-runner
PORT=8084
GRPC_PORT=9084

# Database (requerido)
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=code_runner_db

# Kafka / Azure Event Hub (requerido)
KAFKA_BOOTSTRAP_SERVERS=tu-namespace.servicebus.windows.net:9093
KAFKA_CONNECTION_STRING=Endpoint=sb://tu-namespace.servicebus.windows.net/;SharedAccessKeyName=RootManageSharedAccessKey;SharedAccessKey=tu-clave==

# Service Discovery (opcional)
SERVICE_DISCOVERY_URL=http://eureka:8761/eureka
SERVICE_NAME=CODE-RUNNER-SERVICE
```

**Total: 11 líneas de configuración**

---

## 🎯 Comparación

### Antes vs Ahora

| Categoría | Antes | Ahora |
|-----------|-------|-------|
| **Variables totales** | 31 | 11 |
| **Variables requeridas** | 18 | 6 |
| **Líneas en .env** | ~150 | ~30 |
| **Complejidad** | Alta | Baja |
| **Tópicos Kafka** | Hardcoded | Dinámicos |
| **Service Discovery** | Manual | Auto-activado |

### Reducción

- ✅ **64% menos variables** (de 31 a 11)
- ✅ **67% menos configuración requerida** (de 18 a 6)
- ✅ **80% menos líneas** (de ~150 a ~30)

---

## 📊 Valores Hardcoded en Código

### ubicación: `env/variables.go`

```go
// Logging
Level:  "info",
Format: "json",

// Kafka
Topic:             "challenge.completed",
ConsumerGroup:     "code-runner-service",
SASLMechanism:     "PLAIN",
SecurityProtocol:  "SASL_SSL",
ProducerTimeoutMs: 30000,
ConsumerTimeoutMs: 30000,
MaxRetries:        3,

// Database
SSLMode:         "disable",
Timezone:        "UTC",
MaxOpenConns:    25,
MaxIdleConns:    10,
ConnMaxLifetime: 3600 * time.Second,
```

### ¿Por qué hardcoded?

1. **Valores estándar** - No necesitan cambiar entre entornos
2. **Seguridad** - Configuración SASL_SSL correcta siempre
3. **Simplicidad** - Menos configuración = menos errores
4. **Best practices** - Valores optimizados por defecto

---

## 🔍 Validación

### Comando de Validación

```bash
# Validar configuración simplificada
make validate

# O directamente
./scripts/validate-config.sh
```

### Qué Valida

El script ahora valida:
- ✅ **6 variables esenciales** (DB y Kafka)
- ✅ **Formato de connection strings**
- ✅ **Sin EntityPath** en Kafka (ya no requerido)
- ✅ **Auto-detección de Azure**

Lo que **NO valida** (porque está hardcoded):
- ❌ SASL settings
- ❌ Timeout settings
- ❌ Connection pool settings
- ❌ Log settings

---

## 🚀 Deployment Rápido

### 3 Pasos

```bash
# 1. Configurar (solo 6 valores esenciales)
cp .env.example .env
nano .env

# 2. Validar
make validate

# 3. Desplegar
make deploy
```

---

## 📚 Documentación Relacionada

- **[.env.example](.env.example)** - Template actualizado
- **[docs/KAFKA_DYNAMIC_TOPICS.md](docs/KAFKA_DYNAMIC_TOPICS.md)** - Guía de tópicos dinámicos
- **[QUICK_START.md](QUICK_START.md)** - Inicio rápido
- **[INSTRUCCIONES_DESPLIEGUE.md](INSTRUCCIONES_DESPLIEGUE.md)** - Instrucciones completas

---

## ✅ Beneficios de la Simplificación

1. **Más Fácil de Configurar**
   - Solo 6 valores esenciales
   - Menos errores de configuración
   - Onboarding más rápido

2. **Más Flexible**
   - Tópicos dinámicos
   - Sin reconfiguracion para nuevos tópicos
   - Multi-tenant ready

3. **Más Seguro**
   - SASL_SSL siempre correcto
   - No hay forma de configurarlo mal
   - Valores optimizados

4. **Más Mantenible**
   - Menos archivos de configuración
   - Cambios centralizados en código
   - Menos documentación que mantener

---

## 🎉 Resumen

**Configuración Simplificada:**
- ✅ Solo 11 variables en .env (antes 31)
- ✅ Solo 6 requeridas (antes 18)
- ✅ Tópicos Kafka dinámicos
- ✅ Service Discovery auto-activado
- ✅ Valores optimizados hardcoded
- ✅ 80% menos configuración

**¡Listo para usar!** 🚀

---

**Última Actualización:** 2025  
**Versión:** 2.0.0 (Configuración Simplificada)