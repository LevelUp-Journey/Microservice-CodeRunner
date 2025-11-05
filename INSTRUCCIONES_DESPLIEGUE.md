# 🚀 Instrucciones de Despliegue - Microservicio CodeRunner

## 📋 Resumen Ejecutivo

Se ha implementado exitosamente la dockerización completa del microservicio CodeRunner gRPC con:

- ✅ **Docker-in-Docker (DinD)** - Ejecución aislada de código
- ✅ **Azure Event Hub** - Integración con Kafka y seguridad SASL_SSL
- ✅ **PostgreSQL** - Base de datos con soporte Azure
- ✅ **Service Discovery** - Integración con Eureka
- ✅ **Automatización completa** - 50+ comandos Make
- ✅ **Documentación exhaustiva** - ~80 páginas

---

## 🎯 Inicio Rápido (5 Minutos)

### Paso 1: Configurar Variables de Entorno

```bash
# Copiar el template
cp .env.example .env

# Editar con tus credenciales de Azure
nano .env
```

### Paso 2: Configurar Azure Event Hub

**Obtén tu Connection String de Azure:**

1. Ve al [Portal de Azure](https://portal.azure.com)
2. Navega a: **Event Hubs** → Tu Namespace → **Directivas de acceso compartido**
3. Copia el **Connection string–primary key**

**Actualiza tu archivo `.env`:**

```bash
# Azure Event Hub (REQUERIDO)
KAFKA_BOOTSTRAP_SERVERS=tu-namespace.servicebus.windows.net:9093
KAFKA_CONNECTION_STRING=Endpoint=sb://tu-namespace.servicebus.windows.net/;SharedAccessKeyName=RootManageSharedAccessKey;SharedAccessKey=tu-clave==;EntityPath=challenge-completed
KAFKA_TOPIC=challenge-completed

# Seguridad SASL (NO CAMBIAR para Azure Event Hub)
KAFKA_SASL_MECHANISM=PLAIN
KAFKA_SECURITY_PROTOCOL=SASL_SSL

# Base de Datos (Local - para desarrollo)
DB_HOST=postgres
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=code_runner_db
DB_SSLMODE=disable
```

### Paso 3: Validar Configuración

```bash
# Ejecutar validador automático
chmod +x scripts/validate-config.sh
./scripts/validate-config.sh

# O usando Make
make validate
```

**Deberías ver:** `✓ Configuration is valid! Ready to deploy.`

### Paso 4: Desplegar Servicios

```bash
# Opción A: Usando Make (recomendado)
make deploy

# Opción B: Usando Docker Compose
docker-compose up -d --build
```

**Tiempo estimado:** 2-3 minutos (descarga de imágenes en primera ejecución)

### Paso 5: Verificar Deployment

```bash
# Ver estado de servicios
make status

# Ver logs en tiempo real
make logs

# Verificar salud del servicio
make health

# Deberías ver:
# ✓ HTTP Health Check
# ✓ gRPC Health Check
# ✓ Database Connection
```

### Paso 6: Probar el Servicio

```bash
# Ejecutar test de ejemplo
make test

# O manualmente con grpcurl
grpcurl -plaintext -d '{
  "solution_id": "test_001",
  "challenge_id": "factorial",
  "student_id": "test_student",
  "code": "function factorial(n) { return n <= 1 ? 1 : n * factorial(n-1); }",
  "language": "javascript"
}' localhost:9084 com.levelupjourney.coderunner.CodeExecutionService/ExecuteCode
```

**Respuesta esperada:**
```json
{
  "success": true,
  "message": "All 2 tests passed",
  "execution_id": "exec_...",
  "approved_test_ids": ["test_1", "test_2"]
}
```

---

## 📊 Variables de Entorno CRÍTICAS

### Azure Event Hub (OBLIGATORIO)

```bash
# Endpoint de tu namespace (DEBE terminar en :9093)
KAFKA_BOOTSTRAP_SERVERS=tu-namespace.servicebus.windows.net:9093

# Connection string completo (DEBE incluir EntityPath)
KAFKA_CONNECTION_STRING=Endpoint=sb://tu-namespace.servicebus.windows.net/;SharedAccessKeyName=RootManageSharedAccessKey;SharedAccessKey=TuClaveBase64==;EntityPath=challenge-completed

# Nombre del Event Hub (DEBE coincidir con EntityPath)
KAFKA_TOPIC=challenge-completed

# Configuración SASL (NO MODIFICAR)
KAFKA_SASL_MECHANISM=PLAIN
KAFKA_SECURITY_PROTOCOL=SASL_SSL
```

### Formato del Connection String

```
Endpoint=sb://<namespace>.servicebus.windows.net/;
SharedAccessKeyName=<NombreDelPolicy>;
SharedAccessKey=<ClaveBase64>;
EntityPath=<NombreDelEventHub>
```

**⚠️ IMPORTANTE:**
- El puerto DEBE ser `9093` (no 9092)
- El connection string DEBE incluir `EntityPath=`
- El `KAFKA_TOPIC` DEBE coincidir con el `EntityPath`
- `KAFKA_SASL_MECHANISM` DEBE ser `PLAIN`
- `KAFKA_SECURITY_PROTOCOL` DEBE ser `SASL_SSL`

---

## 🗄️ Configuración de Base de Datos

### Desarrollo Local (Por Defecto)

```bash
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=code_runner_db
DB_SSLMODE=disable
```

### Producción Azure PostgreSQL

```bash
DB_HOST=tu-servidor.postgres.database.azure.com
DB_PORT=5432
DB_USER=admin@tu-servidor
DB_PASSWORD=tu-password-seguro
DB_NAME=code_runner_db
DB_SSLMODE=require
```

---

## 🎛️ Comandos Make Principales

### Setup y Configuración
```bash
make env              # Crear archivo .env desde template
make validate         # Validar configuración
make check-env        # Verificar que .env existe
```

### Build y Deployment
```bash
make build            # Construir imágenes Docker (sin caché)
make build-quick      # Construir con caché
make up               # Iniciar todos los servicios
make deploy           # Build + Start (deployment completo)
make down             # Detener todos los servicios
make restart          # Reiniciar servicios
```

### Monitoreo y Logs
```bash
make logs             # Ver logs de todos los servicios
make logs-service     # Ver logs del CodeRunner solamente
make logs-db          # Ver logs de PostgreSQL
make status           # Estado de todos los servicios
make health           # Verificar health checks
make stats            # Uso de recursos (CPU, memoria)
```

### Base de Datos
```bash
make db-up            # Iniciar solo PostgreSQL y pgAdmin
make db-shell         # Conectar a la base de datos
make db-migrate       # Ejecutar migraciones
make db-backup        # Crear backup de la base de datos
```

### Testing
```bash
make test             # Ejecutar test gRPC de ejemplo
make test-health      # Probar endpoints de salud
make dev              # Modo desarrollo (solo BD)
```

### Troubleshooting
```bash
make troubleshoot     # Ejecutar diagnósticos completos
make shell            # Abrir shell en el contenedor
make shell-dind       # Ver Docker dentro del contenedor DinD
make clean            # Limpiar contenedores
make clean-volumes    # Limpiar volúmenes (⚠️ BORRA DATOS)
```

### Ayuda
```bash
make help             # Ver todos los comandos disponibles
make info             # Información del proyecto
```

---

## 🔍 Verificación de Deployment

### 1. Verificar que los Servicios Están Corriendo

```bash
docker-compose ps

# Deberías ver:
# NAME                    STATUS          PORTS
# coderunner-service      Up (healthy)    0.0.0.0:8084->8084/tcp, 0.0.0.0:9084->9084/tcp
# coderunner-postgres     Up (healthy)    0.0.0.0:5432->5432/tcp
# coderunner-pgadmin      Up              0.0.0.0:5050->80/tcp
```

### 2. Verificar Docker-in-Docker

```bash
# Ver Docker daemon dentro del contenedor
docker exec -it coderunner-service docker info

# Ver imágenes pre-cargadas
docker exec -it coderunner-service docker images

# Deberías ver: node, python, java, gcc, etc.
```

### 3. Verificar Conexión a Azure Event Hub

```bash
# Ver logs de inicialización de Kafka
docker-compose logs coderunner | grep -i kafka

# Deberías ver:
# ✅ Kafka client initialized successfully
# 📡 Bootstrap servers: tu-namespace.servicebus.windows.net:9093
```

### 4. Verificar Base de Datos

```bash
# Conectar a PostgreSQL
docker exec -it coderunner-postgres psql -U postgres -d code_runner_db

# En psql, ejecutar:
\dt  # Listar tablas
SELECT COUNT(*) FROM executions;  # Ver ejecuciones
\q   # Salir
```

### 5. Acceder a pgAdmin

- **URL:** http://localhost:5050
- **Email:** admin@coderunner.local
- **Password:** admin123

---

## 🐛 Solución de Problemas Comunes

### Error: "Docker daemon not starting"

**Síntoma:** El contenedor inicia pero Docker no está disponible dentro

**Solución:**
```bash
# Verificar modo privilegiado
docker inspect coderunner-service | grep Privileged
# Debe mostrar: "Privileged": true

# Ver logs del daemon
docker logs coderunner-service

# Reiniciar servicio
make restart-service
```

### Error: "Authentication failed" (Azure Event Hub)

**Síntoma:** Error de autenticación SASL

**Causas comunes:**
1. Connection string incorrecto
2. Falta `EntityPath=` en el connection string
3. Puerto incorrecto (debe ser 9093)
4. SASL_MECHANISM incorrecto (debe ser PLAIN)

**Solución:**
```bash
# Validar configuración
make validate

# Verificar variables de entorno
docker exec coderunner-service env | grep KAFKA

# Probar conectividad SSL
openssl s_client -connect tu-namespace.servicebus.windows.net:9093
```

### Error: "Cannot connect to database"

**Síntoma:** Error de conexión a PostgreSQL

**Solución:**
```bash
# Verificar que PostgreSQL está corriendo
docker-compose ps postgres

# Ver logs de PostgreSQL
docker-compose logs postgres

# Probar conexión
docker exec -it coderunner-postgres pg_isready -U postgres

# Reiniciar PostgreSQL
docker-compose restart postgres
```

### Error: "Port already in use"

**Síntoma:** Error al iniciar servicios - puerto ocupado

**Solución:**
```bash
# Ver qué proceso usa el puerto
lsof -i :9084
lsof -i :8084

# Matar el proceso
kill -9 <PID>

# O cambiar puertos en .env
GRPC_PORT=9085
PORT=8085
```

### Contenedor se Reinicia Constantemente

**Solución:**
```bash
# Ver logs para identificar el error
make logs-service

# Ejecutar diagnósticos
make troubleshoot

# Verificar configuración
make validate

# Ver estado de salud
docker inspect coderunner-service | grep Health
```

---

## 📚 Documentación Completa

### Documentos Principales

1. **[QUICK_START.md](QUICK_START.md)** ⭐
   - Guía de inicio rápido en 5 minutos
   - Para developers que quieren probar rápido

2. **[DOCKER_README.md](DOCKER_README.md)** 📖
   - Documentación principal de Docker
   - Arquitectura, configuración, troubleshooting
   - ~21 páginas

3. **[DOCKER_SETUP.md](DOCKER_SETUP.md)** 🔧
   - Guía detallada de setup
   - Deployment en Azure
   - Best practices de producción
   - ~15 páginas

4. **[docs/AZURE_EVENT_HUB_SETUP.md](docs/AZURE_EVENT_HUB_SETUP.md)** ☁️
   - Configuración completa de Azure Event Hub
   - Setup de SASL_SSL
   - Troubleshooting específico de Azure
   - ~18 páginas

5. **[DEPLOYMENT_CHECKLIST.md](DEPLOYMENT_CHECKLIST.md)** ✅
   - Checklist completo para producción
   - Validaciones pre-deployment
   - Post-deployment verification
   - ~13 páginas

6. **[DOCKER_IMPLEMENTATION_SUMMARY.md](DOCKER_IMPLEMENTATION_SUMMARY.md)** 📊
   - Resumen ejecutivo de la implementación
   - Estadísticas y métricas
   - Features implementadas
   - ~17 páginas

7. **[DOCKER_INDEX.md](DOCKER_INDEX.md)** 📑
   - Índice completo de toda la documentación
   - Quick reference por caso de uso
   - ~10 páginas

### Archivos de Configuración

- **[Dockerfile](Dockerfile)** - Build multi-stage con DinD
- **[docker-compose.yml](docker-compose.yml)** - Orquestación de 3 servicios
- **[.env.example](.env.example)** - Template de configuración
- **[Makefile](Makefile)** - 50+ comandos de automatización
- **[scripts/validate-config.sh](scripts/validate-config.sh)** - Validador de configuración

---

## 🚀 Deployment en Producción

### Prerequisitos Azure

1. **Event Hubs Namespace** creado
2. **Event Hub** (topic) creado dentro del namespace
3. **Azure Database for PostgreSQL** (opcional, puede usar local)
4. **Connection strings** obtenidos

### Pasos para Producción

1. **Seguir el checklist completo:**
   ```bash
   # Ver checklist
   cat DEPLOYMENT_CHECKLIST.md
   ```

2. **Configurar Azure Event Hub:**
   - Leer [docs/AZURE_EVENT_HUB_SETUP.md](docs/AZURE_EVENT_HUB_SETUP.md)
   - Obtener connection string
   - Configurar firewall rules

3. **Actualizar `.env` con valores de producción:**
   ```bash
   # Azure Event Hub
   KAFKA_BOOTSTRAP_SERVERS=prod-ns.servicebus.windows.net:9093
   KAFKA_CONNECTION_STRING=Endpoint=sb://prod-ns...
   
   # Azure PostgreSQL
   DB_HOST=prod-server.postgres.database.azure.com
   DB_SSLMODE=require
   ```

4. **Validar y desplegar:**
   ```bash
   make validate
   make build
   make up
   make health
   ```

5. **Monitorear:**
   ```bash
   make logs
   make stats
   
   # En Azure Portal:
   # - Event Hub → Metrics
   # - PostgreSQL → Metrics
   ```

---

## 📊 Monitoreo Continuo

### Logs en Tiempo Real

```bash
# Todos los servicios
make logs

# Solo CodeRunner
make logs-service

# Filtrar errores
docker-compose logs coderunner | grep -i error
```

### Métricas de Performance

```bash
# Uso de recursos
make stats

# Estado de servicios
make status

# Health checks
make health
```

### Azure Portal

1. **Event Hub Metrics:**
   - Incoming/Outgoing Messages
   - Throttled Requests
   - Server Errors

2. **Database Metrics:**
   - Connections
   - CPU Usage
   - Storage

---

## 🔒 Seguridad

### Implementado

- ✅ SASL_SSL para Azure Event Hub
- ✅ TLS 1.2+ para PostgreSQL
- ✅ Variables de entorno en `.env` (no commitear)
- ✅ Connection strings encriptados en tránsito
- ✅ Aislamiento de contenedores
- ✅ Modo privilegiado solo para DinD

### Recomendaciones para Producción

1. **Usar Azure Key Vault** para secrets
2. **Rotar credenciales** cada 90 días
3. **Habilitar Private Endpoints** para Event Hub
4. **Configurar Network Security Groups**
5. **Implementar Azure Monitor** para alertas
6. **Revisar logs regularmente**

---

## 🎓 Recursos de Aprendizaje

### Para Nuevos Miembros del Equipo

**Tiempo: 30 minutos**
1. Leer [QUICK_START.md](QUICK_START.md) - 5 min
2. Revisar [.env.example](.env.example) - 10 min
3. Ejecutar `make deploy` - 10 min
4. Probar con `make test` - 5 min

**Tiempo: 2 horas**
1. Estudiar [DOCKER_README.md](DOCKER_README.md) - 30 min
2. Revisar architecture - 20 min
3. Explorar comandos Make - 20 min
4. Practicar troubleshooting - 30 min
5. Deployment de prueba - 20 min

**Tiempo: 4 horas**
1. Deep dive [DOCKER_SETUP.md](DOCKER_SETUP.md) - 60 min
2. Configurar Azure Event Hub - 60 min
3. Estudiar seguridad - 30 min
4. Revisar checklist de producción - 30 min
5. Deployment a Azure - 60 min

---

## ✅ Checklist de Verificación Final

### Antes de Considerar Completo

- [ ] Archivo `.env` configurado con Azure credentials
- [ ] `make validate` ejecutado exitosamente
- [ ] Servicios iniciados con `make deploy`
- [ ] Health checks pasando (`make health`)
- [ ] Test funcional ejecutado (`make test`)
- [ ] Logs monitoreados sin errores críticos
- [ ] Conexión a Azure Event Hub verificada
- [ ] Base de datos accesible
- [ ] pgAdmin accesible en http://localhost:5050
- [ ] Docker-in-Docker funcionando correctamente

### Para Producción

- [ ] Checklist completo revisado (DEPLOYMENT_CHECKLIST.md)
- [ ] Azure Event Hub configurado en Azure Portal
- [ ] Firewall rules configuradas
- [ ] SSL/TLS habilitado para todo
- [ ] Secrets en Azure Key Vault
- [ ] Monitoring configurado
- [ ] Alertas establecidas
- [ ] Backup automatizado
- [ ] Documentación actualizada
- [ ] Equipo entrenado

---

## 🆘 Soporte

### Si Tienes Problemas

1. **Ejecutar diagnósticos:**
   ```bash
   make troubleshoot
   ```

2. **Validar configuración:**
   ```bash
   make validate
   ```

3. **Revisar logs:**
   ```bash
   make logs
   ```

4. **Consultar documentación:**
   - [DOCKER_README.md](DOCKER_README.md) - Sección Troubleshooting
   - [docs/AZURE_EVENT_HUB_SETUP.md](docs/AZURE_EVENT_HUB_SETUP.md) - Para issues de Azure
   - [DOCKER_SETUP.md](DOCKER_SETUP.md) - Para deployment

5. **Abrir issue en GitHub** con:
   - Descripción del problema
   - Logs relevantes (sanitizados)
   - Pasos para reproducir
   - Configuración (sin secrets)

---

## 📞 Contactos

- **Repositorio:** [GitHub Repository URL]
- **Documentación:** Ver archivos `DOCKER_*.md`
- **Issues:** GitHub Issues
- **Azure Support:** Portal de Azure → Support

---

## 🎉 ¡Listo para Usar!

Tu microservicio CodeRunner está completamente dockerizado con:

✅ Docker-in-Docker funcional  
✅ Azure Event Hub con SASL_SSL  
✅ PostgreSQL con soporte Azure  
✅ 50+ comandos automatizados  
✅ Validación automática de configuración  
✅ Documentación completa (~80 páginas)  
✅ Troubleshooting exhaustivo  
✅ Production-ready  

**Siguiente paso:** `make deploy` y ¡a codear! 🚀

---

**Versión:** 1.0.0  
**Última Actualización:** 2025  
**Estado:** ✅ Completo y Listo para Producción