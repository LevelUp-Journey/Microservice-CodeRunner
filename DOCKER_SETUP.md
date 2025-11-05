# 🐳 Docker Setup Guide - CodeRunner Microservice with DinD

This guide explains how to dockerize and run the CodeRunner gRPC microservice using Docker-in-Docker (DinD) with Azure Event Hub integration.

## 📋 Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Architecture](#architecture)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Azure Event Hub Setup](#azure-event-hub-setup)
- [Azure PostgreSQL Setup](#azure-postgresql-setup)
- [Production Deployment](#production-deployment)
- [Troubleshooting](#troubleshooting)

## 🎯 Overview

This microservice uses a **Docker-in-Docker (DinD)** strategy to execute code in isolated containers. This approach provides:

- ✅ **Complete isolation** between code executions
- ✅ **Security** through containerization
- ✅ **Resource control** and limits
- ✅ **Multi-language support** (JavaScript, Python, Java, C++, C#, Go, Rust, TypeScript)
- ✅ **Azure integration** ready (Event Hub, PostgreSQL, Service Discovery)

## 📦 Prerequisites

### Required Software

- **Docker** 20.10+ with Docker Compose
- **Docker Engine** with privileged mode support
- **Azure Account** (for production deployment)

### Required Accounts

- **Azure Event Hub** namespace and event hub
- **Azure Database for PostgreSQL** (or local PostgreSQL)
- **Azure Service Discovery** (optional, Eureka-compatible)

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────┐
│                  Docker Compose Stack                    │
├─────────────────────────────────────────────────────────┤
│                                                           │
│  ┌─────────────────────────────────────────────────┐   │
│  │         CodeRunner Service (DinD)               │   │
│  │  ┌─────────────────────────────────────────┐   │   │
│  │  │     gRPC Server (Port 9084)             │   │   │
│  │  │     Health Check (Port 8084)            │   │   │
│  │  └─────────────────────────────────────────┘   │   │
│  │                                                  │   │
│  │  ┌─────────────────────────────────────────┐   │   │
│  │  │    Docker Daemon (privileged mode)      │   │   │
│  │  │  ┌────────┐ ┌────────┐ ┌────────┐      │   │   │
│  │  │  │ Node.js│ │ Python │ │  C++   │ ...  │   │   │
│  │  │  └────────┘ └────────┘ └────────┘      │   │   │
│  │  └─────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────┘   │
│                      ↓                                   │
│  ┌─────────────────────────────────────────────────┐   │
│  │         PostgreSQL Database                      │   │
│  │         (Port 5432)                              │   │
│  └─────────────────────────────────────────────────┘   │
│                                                           │
│  ┌─────────────────────────────────────────────────┐   │
│  │         pgAdmin (Port 5050)                      │   │
│  └─────────────────────────────────────────────────┘   │
│                                                           │
└─────────────────────────────────────────────────────────┘
                         ↓
         ┌───────────────────────────────┐
         │    Azure Event Hub (Kafka)    │
         │    (SASL_SSL on Port 9093)    │
         └───────────────────────────────┘
```

## 🚀 Quick Start

### Step 1: Clone and Navigate

```bash
git clone <repository-url>
cd Microservice-CodeRunner
```

### Step 2: Configure Environment

```bash
# Copy the example environment file
cp .env.example .env

# Edit .env with your configuration
nano .env  # or use your favorite editor
```

### Step 3: Start Services

#### Option A: Full Stack (Recommended for Development)

```bash
# Start all services (PostgreSQL, pgAdmin, CodeRunner)
docker-compose up -d

# View logs
docker-compose logs -f coderunner
```

#### Option B: Database Only (for Local Development)

```bash
# Start only PostgreSQL and pgAdmin
docker-compose up -d postgres pgadmin

# Build and run CodeRunner locally
go build -o bin/coderunner ./cmd/server
./bin/coderunner
```

### Step 4: Verify Deployment

```bash
# Check service health
curl http://localhost:8084/health

# Test gRPC with grpcurl
grpcurl -plaintext localhost:9084 list

# View Docker containers inside DinD
docker exec -it coderunner-service docker ps
```

## ⚙️ Configuration

### Environment Variables Reference

#### Application Settings

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `APP_NAME` | Application name | `microservice-code-runner` | No |
| `API_VERSION` | API version | `v1` | No |
| `PORT` | HTTP port for health checks | `8084` | No |
| `GRPC_PORT` | gRPC server port | `9084` | No |

#### Database Settings (PostgreSQL)

| Variable | Description | Example | Required |
|----------|-------------|---------|----------|
| `DB_HOST` | Database hostname | `postgres` or `*.postgres.database.azure.com` | Yes |
| `DB_PORT` | Database port | `5432` | Yes |
| `DB_USER` | Database user | `postgres` or `admin@server` | Yes |
| `DB_PASSWORD` | Database password | `your-password` | Yes |
| `DB_NAME` | Database name | `code_runner_db` | Yes |
| `DB_SSLMODE` | SSL mode | `disable` (local) or `require` (Azure) | No |
| `DB_TIMEZONE` | Database timezone | `UTC` | No |

#### Kafka / Azure Event Hub Settings

| Variable | Description | Example | Required |
|----------|-------------|---------|----------|
| `KAFKA_BOOTSTRAP_SERVERS` | Event Hub endpoint | `namespace.servicebus.windows.net:9093` | Yes |
| `KAFKA_CONNECTION_STRING` | Full connection string | `Endpoint=sb://...;SharedAccessKeyName=...` | Yes |
| `KAFKA_TOPIC` | Event Hub name | `challenge.completed` | Yes |
| `KAFKA_CONSUMER_GROUP` | Consumer group ID | `code-runner-service` | No |
| `KAFKA_SASL_MECHANISM` | SASL mechanism | `PLAIN` | No |
| `KAFKA_SECURITY_PROTOCOL` | Security protocol | `SASL_SSL` | No |
| `KAFKA_PRODUCER_TIMEOUT_MS` | Producer timeout | `30000` | No |
| `KAFKA_CONSUMER_TIMEOUT_MS` | Consumer timeout | `30000` | No |
| `KAFKA_MAX_RETRIES` | Max retry attempts | `3` | No |

#### Service Discovery Settings (Optional)

| Variable | Description | Example | Required |
|----------|-------------|---------|----------|
| `SERVICE_DISCOVERY_ENABLED` | Enable/disable Eureka | `false` | No |
| `SERVICE_DISCOVERY_URL` | Eureka server URL | `http://eureka:8761/eureka` | No |
| `SERVICE_NAME` | Service name in Eureka | `CODE-RUNNER-SERVICE` | No |
| `SERVICE_PUBLIC_IP` | Public IP (auto-detect if empty) | `20.30.40.50` | No |

## ☁️ Azure Event Hub Setup

### Step 1: Create Event Hub Namespace

1. Go to **Azure Portal** → **Event Hubs**
2. Click **Create** → **Event Hubs Namespace**
3. Fill in the details:
   - **Name**: `your-namespace` (must be unique)
   - **Pricing Tier**: Standard or Premium
   - **Region**: Choose your region

### Step 2: Create Event Hub (Topic)

1. Navigate to your Event Hub namespace
2. Click **+ Event Hub**
3. Name: `challenge.completed` (or your custom name)
4. Partition Count: `2-4` (based on expected load)
5. Message Retention: `1-7 days`

### Step 3: Get Connection String

1. In your namespace, go to **Shared access policies**
2. Click **RootManageSharedAccessKey** (or create a new policy)
3. Copy the **Connection string–primary key**

### Step 4: Configure Environment Variables

```bash
# In your .env file:
KAFKA_BOOTSTRAP_SERVERS=your-namespace.servicebus.windows.net:9093
KAFKA_CONNECTION_STRING=Endpoint=sb://your-namespace.servicebus.windows.net/;SharedAccessKeyName=RootManageSharedAccessKey;SharedAccessKey=your-key;EntityPath=challenge.completed
KAFKA_TOPIC=challenge.completed
```

### Connection String Format

```
Endpoint=sb://<namespace>.servicebus.windows.net/;
SharedAccessKeyName=<KeyName>;
SharedAccessKey=<KeyValue>;
EntityPath=<EventHubName>
```

### Important Notes

- **Port 9093**: Azure Event Hub uses port 9093 for Kafka protocol
- **SASL_SSL**: Always use `SASL_SSL` security protocol
- **PLAIN mechanism**: Azure Event Hub requires `PLAIN` SASL mechanism
- **Username**: Use `$ConnectionString` as username
- **Password**: Use the full connection string as password

## 🗄️ Azure PostgreSQL Setup

### Step 1: Create Azure Database for PostgreSQL

1. Go to **Azure Portal** → **Azure Database for PostgreSQL**
2. Choose **Flexible Server** (recommended)
3. Fill in the details:
   - **Server name**: `your-server` (must be unique)
   - **Region**: Same as your app
   - **PostgreSQL version**: 15 or 16
   - **Compute + Storage**: Configure based on needs

### Step 2: Configure Firewall

1. Navigate to your PostgreSQL server
2. Go to **Networking** → **Firewall rules**
3. Add your IP address or allow Azure services

### Step 3: Create Database

```sql
-- Connect to your Azure PostgreSQL server
CREATE DATABASE code_runner_db;

-- Grant permissions
GRANT ALL PRIVILEGES ON DATABASE code_runner_db TO your_admin;
```

### Step 4: Configure Environment Variables

```bash
# In your .env file:
DB_HOST=your-server.postgres.database.azure.com
DB_PORT=5432
DB_USER=your_admin@your-server
DB_PASSWORD=your-secure-password
DB_NAME=code_runner_db
DB_SSLMODE=require
```

### Connection String Format

```
host=<server>.postgres.database.azure.com
port=5432
user=<admin>@<server>
password=<password>
dbname=code_runner_db
sslmode=require
```

## 🚢 Production Deployment

### Docker Build

```bash
# Build the Docker image
docker build -t coderunner-service:latest .

# Tag for registry
docker tag coderunner-service:latest your-registry.azurecr.io/coderunner-service:latest

# Push to Azure Container Registry
docker push your-registry.azurecr.io/coderunner-service:latest
```

### Azure Container Instances (ACI)

```bash
# Create resource group
az group create --name coderunner-rg --location eastus

# Create container instance with privileged mode
az container create \
  --resource-group coderunner-rg \
  --name coderunner-aci \
  --image your-registry.azurecr.io/coderunner-service:latest \
  --cpu 2 \
  --memory 4 \
  --registry-login-server your-registry.azurecr.io \
  --registry-username <username> \
  --registry-password <password> \
  --environment-variables \
    APP_NAME=microservice-code-runner \
    GRPC_PORT=9084 \
    PORT=8084 \
  --secure-environment-variables \
    DB_PASSWORD=<password> \
    KAFKA_CONNECTION_STRING=<connection-string> \
  --ports 9084 8084 \
  --dns-name-label coderunner-service
```

### Azure Kubernetes Service (AKS)

Create a deployment YAML:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: coderunner-deployment
spec:
  replicas: 3
  selector:
    matchLabels:
      app: coderunner
  template:
    metadata:
      labels:
        app: coderunner
    spec:
      containers:
      - name: coderunner
        image: your-registry.azurecr.io/coderunner-service:latest
        ports:
        - containerPort: 9084
          name: grpc
        - containerPort: 8084
          name: http
        securityContext:
          privileged: true
        env:
        - name: APP_NAME
          value: "microservice-code-runner"
        - name: DB_HOST
          valueFrom:
            secretKeyRef:
              name: coderunner-secrets
              key: db-host
        # Add more environment variables...
        resources:
          requests:
            memory: "2Gi"
            cpu: "1000m"
          limits:
            memory: "4Gi"
            cpu: "2000m"
---
apiVersion: v1
kind: Service
metadata:
  name: coderunner-service
spec:
  type: LoadBalancer
  ports:
  - port: 9084
    targetPort: 9084
    protocol: TCP
    name: grpc
  - port: 8084
    targetPort: 8084
    protocol: TCP
    name: http
  selector:
    app: coderunner
```

### Security Best Practices

1. **Secrets Management**: Use Azure Key Vault for sensitive data
2. **Managed Identities**: Use Azure Managed Identities instead of connection strings
3. **Network Security**: Configure Network Security Groups (NSG)
4. **SSL/TLS**: Always use SSL for database and SASL_SSL for Event Hub
5. **Monitoring**: Set up Azure Monitor and Application Insights
6. **Resource Limits**: Configure proper CPU and memory limits

## 🔧 Troubleshooting

### Docker-in-Docker Issues

**Problem**: Docker daemon not starting inside container

```bash
# Check if privileged mode is enabled
docker inspect coderunner-service | grep Privileged

# View Docker daemon logs
docker exec -it coderunner-service cat /var/log/docker.log

# Manually start Docker daemon (for debugging)
docker exec -it coderunner-service sh
dockerd-entrypoint.sh &
```

**Solution**: Ensure `privileged: true` is set in docker-compose.yml

### Azure Event Hub Connection Issues

**Problem**: Cannot connect to Event Hub

```bash
# Test connection with openssl
openssl s_client -connect your-namespace.servicebus.windows.net:9093

# Check environment variables
docker exec -it coderunner-service env | grep KAFKA
```

**Common Causes**:
- Incorrect connection string format
- Missing EntityPath in connection string
- Firewall blocking port 9093
- Invalid SASL credentials

**Solution**: Verify connection string format and network connectivity

### PostgreSQL Connection Issues

**Problem**: Database connection refused

```bash
# Test database connection
docker exec -it coderunner-service sh
apk add postgresql-client
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME
```

**Common Causes**:
- Incorrect host format for Azure PostgreSQL
- Firewall rules not configured
- SSL mode mismatch
- Invalid credentials

**Solution**: Check firewall rules and SSL settings

### Memory Issues

**Problem**: Container running out of memory

```bash
# Check memory usage
docker stats coderunner-service

# View container logs
docker logs coderunner-service --tail 100
```

**Solution**: Increase memory limits in docker-compose.yml:

```yaml
coderunner:
  deploy:
    resources:
      limits:
        memory: 8G
      reservations:
        memory: 4G
```

### Image Pull Issues

**Problem**: Cannot pull language runtime images

```bash
# Check Docker images inside DinD
docker exec -it coderunner-service docker images

# Manually pull images
docker exec -it coderunner-service docker pull node:20-alpine
```

**Solution**: Ensure internet connectivity and Docker Hub access

## 📊 Monitoring

### Health Checks

```bash
# HTTP health check
curl http://localhost:8084/health

# gRPC health check
grpcurl -plaintext localhost:9084 \
  com.levelupjourney.coderunner.CodeExecutionService/HealthCheck
```

### Logs

```bash
# View all logs
docker-compose logs -f

# View specific service logs
docker-compose logs -f coderunner

# View Docker daemon logs inside DinD
docker exec -it coderunner-service cat /var/log/docker.log
```

### Metrics

```bash
# Container stats
docker stats coderunner-service

# Docker disk usage
docker exec -it coderunner-service docker system df
```

## 🛠️ Maintenance

### Clean Up Unused Containers

```bash
# Inside the DinD container
docker exec -it coderunner-service docker system prune -af

# Remove old volumes
docker exec -it coderunner-service docker volume prune -f
```

### Update Images

```bash
# Pull latest images
docker-compose pull

# Rebuild and restart
docker-compose up -d --build
```

### Database Migrations

```bash
# Run migrations
docker exec -it coderunner-postgres psql -U postgres -d code_runner_db -f /migrations/001_create_tables.sql
```

## 📚 Additional Resources

- [Docker-in-Docker Best Practices](https://docs.docker.com/engine/security/rootless/)
- [Azure Event Hub Kafka Protocol](https://docs.microsoft.com/azure/event-hubs/event-hubs-for-kafka-ecosystem-overview)
- [Azure Database for PostgreSQL](https://docs.microsoft.com/azure/postgresql/)
- [gRPC Best Practices](https://grpc.io/docs/guides/performance/)

## 🆘 Support

For issues and questions:
1. Check the [Troubleshooting](#troubleshooting) section
2. Review application logs: `docker-compose logs -f`
3. Check Azure portal for service health
4. Open an issue in the repository

---

**Last Updated**: 2025
**Version**: 1.0.0