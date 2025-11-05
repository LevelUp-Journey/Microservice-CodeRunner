# 🐳 Docker Deployment Guide - CodeRunner Microservice

[![Docker](https://img.shields.io/badge/Docker-20.10+-blue.svg)](https://www.docker.com/)
[![Azure](https://img.shields.io/badge/Azure-Event%20Hub-0089D6.svg)](https://azure.microsoft.com/)
[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8.svg)](https://golang.org/)
[![gRPC](https://img.shields.io/badge/gRPC-Protocol-244c5a.svg)](https://grpc.io/)

Complete guide for deploying the CodeRunner gRPC microservice using Docker-in-Docker (DinD) with Azure Event Hub integration.

---

## 📑 Table of Contents

- [Overview](#-overview)
- [Quick Start](#-quick-start)
- [Architecture](#-architecture)
- [Files Structure](#-files-structure)
- [Configuration](#-configuration)
- [Deployment](#-deployment)
- [Monitoring](#-monitoring)
- [Troubleshooting](#-troubleshooting)
- [Best Practices](#-best-practices)

---

## 🎯 Overview

This microservice uses **Docker-in-Docker (DinD)** to execute code in isolated containers, providing maximum security and isolation for multi-language code execution.

### Key Features

- ✅ **gRPC Server** - High-performance RPC communication on port 9084
- ✅ **Docker-in-Docker** - Isolated code execution environments
- ✅ **Azure Event Hub** - Kafka-compatible message broker with SASL_SSL
- ✅ **PostgreSQL Database** - Complete execution tracking
- ✅ **Multi-Language Support** - JavaScript, Python, Java, C++, C#, Go, Rust, TypeScript
- ✅ **Health Monitoring** - HTTP health check endpoint on port 8084
- ✅ **Service Discovery** - Optional Eureka integration

### Technology Stack

| Component | Technology | Purpose |
|-----------|-----------|---------|
| API Protocol | gRPC | High-performance RPC |
| Code Execution | Docker-in-Docker | Isolated execution |
| Message Broker | Azure Event Hub | Event-driven architecture |
| Database | PostgreSQL 16 | Data persistence |
| Container Runtime | Docker 20.10+ | Containerization |
| Orchestration | Docker Compose | Multi-container management |

---

## 🚀 Quick Start

### Prerequisites

- **Docker** 20.10+ with Docker Compose
- **Azure Account** with Event Hub namespace
- **5 minutes** of setup time

### 3-Step Deployment

#### Step 1: Clone and Configure

```bash
# Clone repository
git clone <repository-url>
cd Microservice-CodeRunner

# Create environment file
cp .env.example .env

# Edit with your Azure credentials
nano .env
```

#### Step 2: Get Azure Event Hub Credentials

1. Go to [Azure Portal](https://portal.azure.com)
2. Navigate to: **Event Hubs** → Your Namespace → **Shared access policies**
3. Copy the **Connection string–primary key**
4. Update `.env` file:

```bash
KAFKA_BOOTSTRAP_SERVERS=your-namespace.servicebus.windows.net:9093
KAFKA_CONNECTION_STRING=Endpoint=sb://...;SharedAccessKeyName=...;SharedAccessKey=...;EntityPath=...
KAFKA_TOPIC=challenge-completed
```

#### Step 3: Deploy

```bash
# Validate configuration
make validate

# Deploy services
make deploy

# View logs
make logs
```

### Verify Deployment

```bash
# Check health
curl http://localhost:8084/health

# Test gRPC
grpcurl -plaintext -d '{
  "solution_id": "test_001",
  "challenge_id": "factorial",
  "student_id": "test_student",
  "code": "function factorial(n) { return n <= 1 ? 1 : n * factorial(n-1); }",
  "language": "javascript"
}' localhost:9084 com.levelupjourney.coderunner.CodeExecutionService/ExecuteCode
```

---

## 🏗️ Architecture

### System Architecture

```
┌─────────────────────────────────────────────────────────┐
│                  Docker Compose Stack                    │
├─────────────────────────────────────────────────────────┤
│                                                           │
│  ┌─────────────────────────────────────────────────┐   │
│  │      CodeRunner Service (DinD Container)        │   │
│  │  ┌─────────────────────────────────────────┐   │   │
│  │  │     gRPC Server :9084                   │   │   │
│  │  │     HTTP Health :8084                   │   │   │
│  │  └─────────────────────────────────────────┘   │   │
│  │                                                  │   │
│  │  ┌─────────────────────────────────────────┐   │   │
│  │  │    Docker Daemon (privileged mode)      │   │   │
│  │  │  ┌────────┐ ┌────────┐ ┌────────┐      │   │   │
│  │  │  │Node.js │ │Python  │ │  C++   │      │   │   │
│  │  │  │:20     │ │:3.12   │ │:13.2   │      │   │   │
│  │  │  └────────┘ └────────┘ └────────┘      │   │   │
│  │  │  ┌────────┐ ┌────────┐ ┌────────┐      │   │   │
│  │  │  │ Java   │ │  C#    │ │  Go    │      │   │   │
│  │  │  │:21     │ │:8.0    │ │:1.23   │      │   │   │
│  │  │  └────────┘ └────────┘ └────────┘      │   │   │
│  │  └─────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────┘   │
│                      ↓ :5432                            │
│  ┌─────────────────────────────────────────────────┐   │
│  │         PostgreSQL Database                      │   │
│  │         - Execution Records                      │   │
│  │         - Step Tracking                          │   │
│  │         - Test Results                           │   │
│  └─────────────────────────────────────────────────┘   │
│                                                           │
│  ┌─────────────────────────────────────────────────┐   │
│  │         pgAdmin :5050                            │   │
│  │         Database Management UI                   │   │
│  └─────────────────────────────────────────────────┘   │
│                                                           │
└─────────────────────────────────────────────────────────┘
                         ↓ SASL_SSL :9093
         ┌───────────────────────────────┐
         │    Azure Event Hub (Kafka)    │
         │    - Message Publishing       │
         │    - Event Streaming           │
         └───────────────────────────────┘
```

### Docker-in-Docker Flow

```
User Request (gRPC)
    ↓
CodeRunner Service
    ↓
Creates Isolated Container
    ↓
Executes User Code
    ↓
Captures Output
    ↓
Publishes Event to Azure Event Hub
    ↓
Stores Results in PostgreSQL
    ↓
Returns Response to User
```

---

## 📂 Files Structure

### Docker Configuration Files

```
Microservice-CodeRunner/
├── Dockerfile                      # Multi-stage build with DinD support
├── docker-compose.yml              # Multi-container orchestration
├── .dockerignore                   # Build optimization
├── .env.example                    # Environment template
├── Makefile                        # Automation commands
│
├── scripts/
│   └── validate-config.sh          # Configuration validator
│
├── docs/
│   ├── DOCKER_SETUP.md            # Complete Docker guide
│   └── AZURE_EVENT_HUB_SETUP.md   # Azure Event Hub setup
│
├── QUICK_START.md                  # Quick deployment guide
├── DEPLOYMENT_CHECKLIST.md         # Production checklist
└── DOCKER_README.md                # This file
```

### Key Files Explained

#### `Dockerfile`
- **Multi-stage build** for optimized image size
- **Stage 1**: Build Go binary (golang:1.25.1-alpine)
- **Stage 2**: Runtime with DinD support (docker:27-dind)
- **Features**: Pre-pulls language images, health checks, non-root user

#### `docker-compose.yml`
- **3 services**: CodeRunner, PostgreSQL, pgAdmin
- **Networks**: Bridge network with custom subnet
- **Volumes**: Persistent data for database and DinD storage
- **Health checks**: All services monitored

#### `.env.example`
- **Complete configuration template**
- **Azure Event Hub settings** with SASL_SSL
- **Database connection strings**
- **Service discovery configuration**

#### `Makefile`
- **50+ commands** for common operations
- **Color-coded output** for better UX
- **Validation**, deployment, monitoring, cleanup

---

## ⚙️ Configuration

### Environment Variables

#### Required Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `KAFKA_BOOTSTRAP_SERVERS` | Event Hub endpoint | `ns.servicebus.windows.net:9093` |
| `KAFKA_CONNECTION_STRING` | Full connection string | `Endpoint=sb://...;SharedAccessKey=...` |
| `KAFKA_TOPIC` | Event Hub name | `challenge-completed` |
| `DB_HOST` | Database hostname | `postgres` or `*.postgres.database.azure.com` |
| `DB_USER` | Database username | `postgres` or `admin@server` |
| `DB_PASSWORD` | Database password | `secure-password` |
| `DB_NAME` | Database name | `code_runner_db` |

#### Azure Event Hub Configuration

**Critical Settings for SASL_SSL:**

```bash
# Event Hub endpoint (must end with :9093)
KAFKA_BOOTSTRAP_SERVERS=your-namespace.servicebus.windows.net:9093

# Full connection string (must include EntityPath)
KAFKA_CONNECTION_STRING=Endpoint=sb://your-namespace.servicebus.windows.net/;SharedAccessKeyName=RootManageSharedAccessKey;SharedAccessKey=YourKey==;EntityPath=challenge-completed

# Topic must match EntityPath
KAFKA_TOPIC=challenge-completed

# SASL settings (DO NOT CHANGE for Azure Event Hub)
KAFKA_SASL_MECHANISM=PLAIN
KAFKA_SECURITY_PROTOCOL=SASL_SSL
```

#### Database Configuration

**Local Development:**
```bash
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=code_runner_db
DB_SSLMODE=disable
```

**Azure PostgreSQL:**
```bash
DB_HOST=myserver.postgres.database.azure.com
DB_PORT=5432
DB_USER=admin@myserver
DB_PASSWORD=secure-password
DB_NAME=code_runner_db
DB_SSLMODE=require
```

### Configuration Validation

```bash
# Run validator before deployment
./scripts/validate-config.sh

# Or using Make
make validate
```

**Validator checks:**
- ✅ All required variables present
- ✅ Connection string format
- ✅ EntityPath matches topic
- ✅ Port numbers valid
- ✅ SASL settings correct
- ✅ Azure-specific configurations

---

## 🚀 Deployment

### Development Deployment

```bash
# 1. Setup environment
cp .env.example .env
nano .env  # Edit with your values

# 2. Validate configuration
make validate

# 3. Deploy all services
make deploy

# 4. View logs
make logs

# 5. Check status
make status
```

### Production Deployment

#### Option A: Docker Compose

```bash
# 1. Build production image
make build

# 2. Start services
make up

# 3. Verify health
make health

# 4. Monitor
make stats
```

#### Option B: Azure Container Instances

```bash
# 1. Build and tag
docker build -t coderunner:latest .
docker tag coderunner:latest myregistry.azurecr.io/coderunner:latest

# 2. Login to ACR
az acr login --name myregistry

# 3. Push image
docker push myregistry.azurecr.io/coderunner:latest

# 4. Deploy to ACI
az container create \
  --resource-group coderunner-rg \
  --name coderunner-prod \
  --image myregistry.azurecr.io/coderunner:latest \
  --cpu 2 --memory 4 \
  --environment-variables \
    APP_NAME=coderunner \
    GRPC_PORT=9084 \
  --secure-environment-variables \
    KAFKA_CONNECTION_STRING=$KAFKA_CONNECTION_STRING \
    DB_PASSWORD=$DB_PASSWORD \
  --ports 9084 8084
```

#### Option C: Azure Kubernetes Service

```bash
# 1. Create AKS cluster
az aks create \
  --resource-group coderunner-rg \
  --name coderunner-cluster \
  --node-count 3 \
  --enable-addons monitoring

# 2. Get credentials
az aks get-credentials \
  --resource-group coderunner-rg \
  --name coderunner-cluster

# 3. Deploy
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml

# 4. Check status
kubectl get pods
kubectl get services
```

### Useful Make Commands

```bash
# Setup and Configuration
make env              # Create .env from example
make validate         # Validate configuration
make check-env        # Check if .env exists

# Build and Deploy
make build            # Build images (no cache)
make build-quick      # Build with cache
make up               # Start all services
make up-build         # Build and start
make deploy           # Full deployment

# Database
make db-up            # Start only database
make db-shell         # Connect to PostgreSQL
make db-migrate       # Run migrations
make db-backup        # Backup database
make db-restore       # Restore database

# Monitoring
make logs             # View all logs
make logs-service     # CodeRunner logs only
make logs-db          # Database logs only
make status           # Service status
make health           # Health checks
make stats            # Resource usage

# Maintenance
make restart          # Restart all services
make down             # Stop all services
make clean            # Clean containers
make clean-volumes    # Clean data (WARNING)
make clean-dind       # Clean DinD resources

# Testing
make test             # Run test gRPC call
make test-health      # Test health endpoints
make dev              # Dev mode (DB only)

# Troubleshooting
make troubleshoot     # Run diagnostics
make shell            # Shell in container
make shell-dind       # Access Docker in DinD

# Information
make help             # Show all commands
make info             # Project information
make version          # Show versions
```

---

## 📊 Monitoring

### Health Checks

```bash
# HTTP Health Check
curl http://localhost:8084/health

# gRPC Health Check
grpcurl -plaintext localhost:9084 \
  com.levelupjourney.coderunner.CodeExecutionService/HealthCheck
```

### Logs

```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f coderunner

# Last 100 lines
docker-compose logs --tail=100

# Filter by keyword
docker-compose logs coderunner | grep -i error
```

### Metrics

```bash
# Container stats
docker stats coderunner-service

# Docker images inside DinD
docker exec -it coderunner-service docker images

# Docker containers inside DinD
docker exec -it coderunner-service docker ps

# System info
docker exec -it coderunner-service docker system df
```

### Azure Portal Monitoring

1. **Event Hub Metrics**
   - Go to Event Hub → Metrics
   - Monitor: Incoming/Outgoing Messages, Errors, Throttled Requests

2. **Database Metrics**
   - Go to PostgreSQL → Metrics
   - Monitor: Connections, CPU, Memory, Storage

3. **Container Metrics** (if using ACI/AKS)
   - Go to Container Instance → Metrics
   - Monitor: CPU, Memory, Network

---

## 🔧 Troubleshooting

### Common Issues

#### 1. Docker-in-Docker Not Starting

**Symptoms:**
- Container starts but Docker daemon not available
- Error: "Cannot connect to Docker daemon"

**Solution:**
```bash
# Check privileged mode
docker inspect coderunner-service | grep Privileged
# Should show: "Privileged": true

# Check logs
docker logs coderunner-service

# Manually start Docker (debugging)
docker exec -it coderunner-service sh
dockerd-entrypoint.sh &
```

#### 2. Azure Event Hub Connection Failed

**Symptoms:**
- Authentication failed
- Connection timeout
- SSL errors

**Solution:**
```bash
# Validate connection string format
echo $KAFKA_CONNECTION_STRING | grep "EntityPath="

# Test SSL connectivity
openssl s_client -connect your-ns.servicebus.windows.net:9093

# Check environment variables
docker exec coderunner-service env | grep KAFKA

# Verify in Azure Portal
# Event Hub → Shared access policies → Verify key
```

**Common Causes:**
- Missing `EntityPath=` in connection string
- Wrong port (must be 9093, not 9092)
- Incorrect SASL mechanism (must be PLAIN)
- Invalid security protocol (must be SASL_SSL)
- Firewall blocking port 9093

#### 3. Database Connection Issues

**Symptoms:**
- Cannot connect to database
- SSL handshake failed
- Authentication failed

**Solution:**
```bash
# Check database status
docker-compose ps postgres

# Test connection
docker exec -it coderunner-postgres pg_isready -U postgres

# Check logs
docker-compose logs postgres

# Manual connection
docker exec -it coderunner-postgres \
  psql -U postgres -d code_runner_db
```

**For Azure PostgreSQL:**
```bash
# Check firewall rules
az postgres server firewall-rule list \
  --resource-group myRG \
  --server-name myserver

# Add your IP
az postgres server firewall-rule create \
  --resource-group myRG \
  --server-name myserver \
  --name AllowMyIP \
  --start-ip-address 1.2.3.4 \
  --end-ip-address 1.2.3.4
```

#### 4. Port Already in Use

**Symptoms:**
- Error: "bind: address already in use"

**Solution:**
```bash
# Find process using port
lsof -i :9084
# or
netstat -tulpn | grep 9084

# Kill process
kill -9 <PID>

# Or change port in .env
GRPC_PORT=9085
PORT=8085
```

#### 5. Out of Memory

**Symptoms:**
- Container killed
- OOMKilled status
- Service crashes

**Solution:**
```bash
# Check memory usage
docker stats coderunner-service

# Increase memory limit in docker-compose.yml
services:
  coderunner:
    deploy:
      resources:
        limits:
          memory: 8G
        reservations:
          memory: 4G
```

### Debug Mode

```bash
# Enable debug logging
LOG_LEVEL=debug docker-compose up

# View detailed logs
docker-compose logs -f --tail=500 coderunner

# Access container shell
make shell

# Check Docker daemon inside DinD
docker exec -it coderunner-service docker info
```

### Getting Help

1. **Check Documentation**
   - [DOCKER_SETUP.md](DOCKER_SETUP.md) - Complete setup guide
   - [AZURE_EVENT_HUB_SETUP.md](docs/AZURE_EVENT_HUB_SETUP.md) - Event Hub configuration
   - [QUICK_START.md](QUICK_START.md) - Quick start guide

2. **Run Diagnostics**
   ```bash
   make troubleshoot
   ```

3. **Check Configuration**
   ```bash
   make validate
   ```

4. **Review Logs**
   ```bash
   make logs
   ```

---

## 🎯 Best Practices

### Security

1. **Secrets Management**
   - ✅ Use Azure Key Vault for production secrets
   - ✅ Never commit `.env` file to Git
   - ✅ Rotate credentials regularly (every 90 days)
   - ✅ Use Managed Identity when possible

2. **Network Security**
   - ✅ Use Private Endpoints for Event Hub
   - ✅ Configure Network Security Groups
   - ✅ Enable SSL/TLS for all connections
   - ✅ Restrict database access

3. **Container Security**
   - ✅ Run as non-root user where possible
   - ✅ Keep images updated
   - ✅ Scan for vulnerabilities
   - ✅ Use minimal base images

### Performance

1. **Resource Optimization**
   - ✅ Set appropriate CPU/memory limits
   - ✅ Use connection pooling
   - ✅ Enable compression
   - ✅ Optimize database queries

2. **Caching**
   - ✅ Pre-pull Docker images
   - ✅ Use build cache
   - ✅ Cache database connections
   - ✅ Implement result caching

3. **Monitoring**
   - ✅ Set up Application Insights
   - ✅ Configure alerts
   - ✅ Track key metrics
   - ✅ Log important events

### Operations

1. **Deployment**
   - ✅ Use CI/CD pipelines
   - ✅ Test in staging first
   - ✅ Have rollback plan ready
   - ✅ Document changes

2. **Backup**
   - ✅ Regular database backups
   - ✅ Test restore procedures
   - ✅ Store backups securely
   - ✅ Keep multiple versions

3. **Maintenance**
   - ✅ Regular updates
   - ✅ Clean up old images
   - ✅ Monitor disk usage
   - ✅ Review logs regularly

---

## 📚 Additional Resources

### Documentation

- [Main README](README.md) - Project overview
- [Docker Setup Guide](DOCKER_SETUP.md) - Detailed Docker configuration
- [Azure Event Hub Guide](docs/AZURE_EVENT_HUB_SETUP.md) - Event Hub setup with SASL
- [Quick Start](QUICK_START.md) - 5-minute deployment
- [Deployment Checklist](DEPLOYMENT_CHECKLIST.md) - Production deployment

### Official Links

- [Docker Documentation](https://docs.docker.com/)
- [Azure Event Hubs](https://docs.microsoft.com/azure/event-hubs/)
- [Azure PostgreSQL](https://docs.microsoft.com/azure/postgresql/)
- [gRPC Documentation](https://grpc.io/docs/)
- [Go Documentation](https://golang.org/doc/)

### Tools

- [Docker Desktop](https://www.docker.com/products/docker-desktop)
- [Azure CLI](https://docs.microsoft.com/cli/azure/)
- [grpcurl](https://github.com/fullstorydev/grpcurl)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)

---

## 📞 Support

### Issue Reporting

If you encounter issues:

1. Run diagnostics: `make troubleshoot`
2. Check logs: `make logs`
3. Validate config: `make validate`
4. Review documentation
5. Open GitHub issue with:
   - Error messages
   - Configuration (sanitized)
   - Steps to reproduce
   - Environment details

### Community

- **GitHub Issues**: Bug reports and feature requests
- **Stack Overflow**: Tag `coderunner-microservice`
- **Azure Support**: For Azure-specific issues

---

## 📄 License

See [LICENSE](LICENSE) file for details.

---

**Version**: 1.0.0  
**Last Updated**: 2025  
**Maintained By**: DevOps Team

---

Made with ❤️ by the CodeRunner Team