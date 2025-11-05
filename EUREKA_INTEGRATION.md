# Guía de Integración con Eureka Service Discovery

Esta guía explica cómo configurar el microservicio Code Runner para que se comunique correctamente con otros microservicios a través de Eureka Service Discovery en un entorno Docker.

## 🎯 Problema Resuelto

El servicio Go ahora se registra en Eureka usando el **hostname del contenedor** en lugar de una dirección IP, lo que permite la comunicación entre microservicios en Docker usando nombres de servicio (DNS).

## 📋 Configuración Requerida

### 1. Variables de Entorno Críticas

```bash
# Nombre del servicio en Eureka (debe coincidir con el que buscan otros servicios)
SERVICE_NAME=CODE-RUNNER-SERVICE

# Hostname del contenedor (debe ser el nombre del servicio en minúsculas con guiones)
HOSTNAME=code-runner-service

# URL del servidor Eureka
SERVICE_DISCOVERY_URL=http://service-discovery:8761/eureka
```

### 2. Configuración en Docker Compose

```yaml
services:
  coderunner:
    container_name: code-runner-service
    hostname: code-runner-service  # ⚠️ IMPORTANTE: Debe coincidir con HOSTNAME
    networks:
      - your-shared-network  # ⚠️ IMPORTANTE: Misma red que otros microservicios
    environment:
      SERVICE_DISCOVERY_URL: http://service-discovery:8761/eureka
      SERVICE_NAME: CODE-RUNNER-SERVICE
      HOSTNAME: code-runner-service
```

## 🔄 Comparación con Spring Boot

### Registro en Eureka

**Spring Boot (application.yml):**
```yaml
eureka:
  instance:
    hostname: ${HOSTNAME:${spring.application.name}}
    instance-id: ${spring.application.name}:${server.port:8083}
    prefer-ip-address: false
```

**Go (equivalente):**
```go
// El servicio se registra con:
// - hostname: "code-runner-service" (del contenedor)
// - instanceId: "code-runner-service:9084"
// - ipAddr: IP local del contenedor (para referencia)
// - vipAddress: "CODE-RUNNER-SERVICE"
```

## 🌐 Comunicación entre Microservicios

### Desde Spring Boot hacia Code Runner (gRPC)

**Configuración en Spring Boot:**
```yaml
grpc:
  client:
    code-runner:
      address: discovery:///code-runner-service
      negotiation-type: plaintext
      max-inbound-message-size: 8MB
```

**Cómo funciona:**
1. Spring Boot busca `CODE-RUNNER-SERVICE` en Eureka
2. Eureka devuelve el hostname: `code-runner-service`
3. Spring Boot resuelve `code-runner-service` vía DNS de Docker
4. Se conecta al puerto gRPC: `code-runner-service:9084`

### Desde Code Runner hacia otros servicios (HTTP/gRPC)

**Ejemplo de cliente gRPC en Go:**
```go
import (
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

// Conectarse usando el hostname del servicio
conn, err := grpc.Dial(
    "challenges-service:8083",
    grpc.WithTransportCredentials(insecure.NewCredentials()),
)
```

**Ejemplo de cliente HTTP en Go:**
```go
import "net/http"

// Llamar a otro microservicio
resp, err := http.Get("http://iam-service:8081/api/users")
```

## 🐳 Docker Compose Multi-Servicio

### Ejemplo de Configuración Completa

```yaml
version: '3.8'

services:
  # Service Discovery (Eureka)
  service-discovery:
    image: steeltoeoss/eureka-server:latest
    container_name: service-discovery
    hostname: service-discovery
    ports:
      - "8761:8761"
    networks:
      - microservices-network
    environment:
      EUREKA_INSTANCE_HOSTNAME: service-discovery
      EUREKA_CLIENT_REGISTERWITEUREKA: "false"
      EUREKA_CLIENT_FETCHREGISTRY: "false"

  # IAM Service (Spring Boot)
  iam-service:
    image: your-repo/iam-service:latest
    container_name: iam-service
    hostname: iam-service
    ports:
      - "8081:8081"
    networks:
      - microservices-network
    environment:
      SPRING_APPLICATION_NAME: iam-service
      SERVER_PORT: 8081
      SERVICE_DISCOVERY_URL: http://service-discovery:8761/eureka
      HOSTNAME: iam-service
    depends_on:
      - service-discovery

  # Challenges Service (Spring Boot)
  challenges-service:
    image: your-repo/challenges-service:latest
    container_name: challenges-service
    hostname: challenges-service
    ports:
      - "8083:8083"
    networks:
      - microservices-network
    environment:
      SPRING_APPLICATION_NAME: challenges-service
      SERVER_PORT: 8083
      SERVICE_DISCOVERY_URL: http://service-discovery:8761/eureka
      HOSTNAME: challenges-service
    depends_on:
      - service-discovery

  # Code Runner Service (Go)
  code-runner-service:
    build:
      context: ./Microservice-CodeRunner
      dockerfile: Dockerfile
    container_name: code-runner-service
    hostname: code-runner-service
    privileged: true
    ports:
      - "9084:9084"
      - "8084:8084"
    networks:
      - microservices-network
    environment:
      APP_NAME: microservice-code-runner
      GRPC_PORT: 9084
      PORT: 8084
      SERVICE_DISCOVERY_URL: http://service-discovery:8761/eureka
      SERVICE_NAME: CODE-RUNNER-SERVICE
      HOSTNAME: code-runner-service
      DB_HOST: postgres
      DB_PORT: 5432
      # ... otras variables
    depends_on:
      - service-discovery
      - postgres
    volumes:
      - dind-storage:/var/lib/docker

  # PostgreSQL (compartido o individual)
  postgres:
    image: postgres:16-alpine
    container_name: postgres
    hostname: postgres
    ports:
      - "5432:5432"
    networks:
      - microservices-network
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: code_runner_db

networks:
  microservices-network:
    driver: bridge
    name: microservices-network

volumes:
  dind-storage:
    driver: local
```

## ✅ Checklist de Verificación

### 1. Antes de iniciar los servicios

- [ ] Todos los servicios están en la **misma red Docker**
- [ ] El `container_name` coincide con el `hostname`
- [ ] El `HOSTNAME` está configurado como variable de entorno
- [ ] El `SERVICE_DISCOVERY_URL` apunta al servidor Eureka correcto

### 2. Durante el inicio

```bash
# Iniciar en orden:
docker-compose up -d service-discovery
# Esperar 30 segundos
docker-compose up -d postgres
# Esperar 10 segundos
docker-compose up -d code-runner-service
docker-compose up -d challenges-service iam-service
```

### 3. Verificar registro en Eureka

**Accede a:** http://localhost:8761

Deberías ver:
```
Application: CODE-RUNNER-SERVICE
Status: UP (1) - code-runner-service:9084
```

### 4. Verificar logs del Code Runner

```bash
docker logs code-runner-service
```

Deberías ver:
```
🌐 Using hostname: code-runner-service
🔑 Instance ID: code-runner-service:9084
📝 Registering service with name: CODE-RUNNER-SERVICE
✅ Service registered in Eureka as CODE-RUNNER-SERVICE at code-runner-service:9084
💓 Heartbeat sent successfully to Eureka
```

### 5. Verificar comunicación desde otro servicio

```bash
# Desde el contenedor de challenges-service
docker exec -it challenges-service sh
curl http://code-runner-service:8084/health
# Debería responder: {"status":"ok"}
```

## 🔧 Troubleshooting

### Error: "no such host: code-runner-service"

**Causa:** Los servicios no están en la misma red Docker

**Solución:**
```yaml
# Asegúrate de que todos los servicios usen la misma red
networks:
  - microservices-network
```

### Error: "Service not found in Eureka"

**Causa:** El `SERVICE_NAME` no coincide con el que buscan otros servicios

**Solución:** Verifica que el nombre sea exactamente igual (case-sensitive)
```bash
# En Code Runner
SERVICE_NAME=CODE-RUNNER-SERVICE

# En Spring Boot (application.yml)
grpc:
  client:
    code-runner:
      address: discovery:///code-runner-service
```

### Error: "Connection refused"

**Causa:** El puerto no está expuesto o el servicio no está escuchando

**Solución:**
```yaml
# Exponer el puerto gRPC
ports:
  - "9084:9084"

# Verificar con:
docker exec -it code-runner-service netstat -tuln | grep 9084
```

### Servicio se registra con IP en lugar de hostname

**Causa:** El `HOSTNAME` no está configurado o el `hostname` del contenedor es diferente

**Solución:**
```yaml
services:
  coderunner:
    container_name: code-runner-service
    hostname: code-runner-service  # ⚠️ IMPORTANTE
    environment:
      HOSTNAME: code-runner-service  # ⚠️ Debe coincidir
```

## 📊 Monitoreo y Health Checks

### Endpoints de Health Check

```bash
# gRPC Health Check (requiere grpcurl)
grpcurl -plaintext code-runner-service:9084 grpc.health.v1.Health/Check

# HTTP Health Check
curl http://code-runner-service:8084/health
```

### Dashboard de Eureka

Accede a: http://localhost:8761

Aquí verás todos los servicios registrados y su estado en tiempo real.

## 🚀 Despliegue en Producción

### Consideraciones Adicionales

1. **DNS Interno:** En producción (Kubernetes, Docker Swarm), usa el DNS interno del cluster
2. **Load Balancing:** Eureka soporta múltiples instancias del mismo servicio
3. **Health Checks:** Configura health checks más robustos
4. **Timeouts:** Ajusta los timeouts según tu red

### Ejemplo Kubernetes

```yaml
apiVersion: v1
kind: Service
metadata:
  name: code-runner-service
spec:
  selector:
    app: code-runner
  ports:
    - name: grpc
      port: 9084
      targetPort: 9084
    - name: http
      port: 8084
      targetPort: 8084
---
apiVersion: v1
kind: Pod
metadata:
  name: code-runner-service
  labels:
    app: code-runner
spec:
  containers:
  - name: code-runner
    image: your-repo/code-runner:latest
    env:
    - name: HOSTNAME
      valueFrom:
        fieldRef:
          fieldPath: metadata.name
    - name: SERVICE_DISCOVERY_URL
      value: "http://eureka-service:8761/eureka"
```

## 📚 Referencias

- [Eureka REST API](https://github.com/Netflix/eureka/wiki/Eureka-REST-operations)
- [Spring Cloud Netflix](https://spring.io/projects/spring-cloud-netflix)
- [Docker Networking](https://docs.docker.com/network/)
- [gRPC Service Discovery](https://github.com/grpc/grpc/blob/master/doc/naming.md)

## 💡 Tips Importantes

1. **Siempre usa hostnames en Docker**, no IPs
2. **El `container_name` debe coincidir con `hostname`**
3. **Todos los servicios deben estar en la misma red**
4. **El `SERVICE_NAME` en Eureka es case-sensitive**
5. **Los heartbeats se envían cada 30 segundos**
6. **Eureka tarda ~90 segundos en marcar un servicio como DOWN**

---

**✅ Si sigues esta guía, tu microservicio Go se integrará perfectamente con tus servicios Spring Boot a través de Eureka.**