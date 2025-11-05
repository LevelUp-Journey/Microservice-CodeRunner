# 🚀 Deployment Checklist - CodeRunner Microservice

Complete checklist for deploying the CodeRunner gRPC microservice with Docker-in-Docker and Azure Event Hub integration.

## 📋 Pre-Deployment Checklist

### Azure Resources Setup

#### ☐ Event Hubs Namespace
- [ ] Created Event Hubs Namespace in Azure Portal
- [ ] Selected **Standard** tier (minimum for Kafka support)
- [ ] Configured throughput units (start with 1, enable auto-inflate)
- [ ] Noted namespace name: `_______________________`
- [ ] Noted region: `_______________________`

#### ☐ Event Hub (Topic) Creation
- [ ] Created Event Hub within namespace
- [ ] Event Hub name: `_______________________` (default: `challenge-completed`)
- [ ] Partition count: `_______` (recommended: 2-4)
- [ ] Message retention: `_______` days (1-7 for Standard tier)

#### ☐ Shared Access Policy
- [ ] Created custom policy or using RootManageSharedAccessKey
- [ ] Policy has **Send** permission (for producers)
- [ ] Policy has **Listen** permission (for consumers)
- [ ] Copied **Connection string–primary key**
- [ ] Verified EntityPath is included in connection string

#### ☐ Azure Database for PostgreSQL (Optional)
- [ ] Created PostgreSQL Flexible Server
- [ ] Server name: `_______________________`
- [ ] PostgreSQL version: `_______` (15 or 16 recommended)
- [ ] Configured firewall rules
- [ ] Created database: `code_runner_db`
- [ ] Noted admin username: `_______________________`
- [ ] Stored admin password securely

### Local Environment Setup

#### ☐ Software Installation
- [ ] Docker 20.10+ installed
- [ ] Docker Compose installed
- [ ] Docker daemon is running
- [ ] Git installed
- [ ] Make installed (optional, but recommended)
- [ ] grpcurl installed (for testing)

#### ☐ Repository Setup
```bash
# Clone repository
[ ] git clone <repository-url>
[ ] cd Microservice-CodeRunner

# Verify files
[ ] ls -la Dockerfile
[ ] ls -la docker-compose.yml
[ ] ls -la .env.example
[ ] ls -la Makefile
```

### Configuration Files

#### ☐ Environment Variables (.env)
```bash
# Create .env file
[ ] cp .env.example .env

# Edit .env file with actual values
[ ] nano .env  # or your favorite editor
```

**Required Variables to Configure:**

```bash
# Application
[ ] APP_NAME=microservice-code-runner
[ ] GRPC_PORT=9084
[ ] PORT=8084

# Azure Event Hub (CRITICAL)
[ ] KAFKA_BOOTSTRAP_SERVERS=your-namespace.servicebus.windows.net:9093
[ ] KAFKA_CONNECTION_STRING=Endpoint=sb://...;SharedAccessKeyName=...;SharedAccessKey=...;EntityPath=...
[ ] KAFKA_TOPIC=challenge-completed
[ ] KAFKA_SASL_MECHANISM=PLAIN
[ ] KAFKA_SECURITY_PROTOCOL=SASL_SSL

# Database
[ ] DB_HOST=postgres (or Azure PostgreSQL host)
[ ] DB_USER=postgres (or Azure admin@servername)
[ ] DB_PASSWORD=<secure-password>
[ ] DB_NAME=code_runner_db
[ ] DB_SSLMODE=disable (local) or require (Azure)

# Service Discovery (if using)
[ ] SERVICE_DISCOVERY_ENABLED=true/false
[ ] SERVICE_DISCOVERY_URL=http://eureka:8761/eureka
```

#### ☐ Validate Configuration
```bash
# Run validation script
[ ] chmod +x scripts/validate-config.sh
[ ] ./scripts/validate-config.sh

# Should show: "Configuration is valid! Ready to deploy."
```

## 🏗️ Build and Deployment

### Step 1: Build Docker Images

```bash
# Option A: Using Make
[ ] make build

# Option B: Using Docker Compose
[ ] docker-compose build --no-cache

# Verify images
[ ] docker images | grep coderunner
```

### Step 2: Start Services

```bash
# Option A: Using Make
[ ] make up

# Option B: Using Docker Compose
[ ] docker-compose up -d

# Check status
[ ] docker-compose ps
```

**Expected Output:**
```
NAME                    STATUS          PORTS
coderunner-service      Up (healthy)    0.0.0.0:8084->8084/tcp, 0.0.0.0:9084->9084/tcp
coderunner-postgres     Up (healthy)    0.0.0.0:5432->5432/tcp
coderunner-pgadmin      Up              0.0.0.0:5050->80/tcp
```

### Step 3: Verify Docker-in-Docker

```bash
# Check Docker daemon inside container
[ ] docker exec -it coderunner-service docker info

# Should show Docker version and system info

# Check pre-pulled images
[ ] docker exec -it coderunner-service docker images

# Should show: node, python, java, gcc, etc.
```

### Step 4: Health Checks

#### ☐ HTTP Health Check
```bash
[ ] curl http://localhost:8084/health

# Expected: {"status":"healthy"}
```

#### ☐ gRPC Health Check
```bash
[ ] grpcurl -plaintext localhost:9084 \
    com.levelupjourney.coderunner.CodeExecutionService/HealthCheck

# Expected: JSON response with "healthy" status
```

#### ☐ Database Connection
```bash
[ ] docker exec -it coderunner-postgres pg_isready -U postgres

# Expected: "postgres:5432 - accepting connections"
```

#### ☐ Azure Event Hub Connection
```bash
# Check logs for Kafka initialization
[ ] docker-compose logs coderunner | grep -i kafka

# Should see:
# ✅ Kafka client initialized successfully
# 📡 Bootstrap servers: your-namespace.servicebus.windows.net:9093
```

### Step 5: Functional Testing

#### ☐ Execute Code Test
```bash
[ ] grpcurl -plaintext -d '{
  "solution_id": "test_deploy_001",
  "challenge_id": "factorial",
  "student_id": "deployment_test",
  "code": "function factorial(n) { return n <= 1 ? 1 : n * factorial(n-1); }",
  "language": "javascript"
}' localhost:9084 com.levelupjourney.coderunner.CodeExecutionService/ExecuteCode

# Expected: success=true, approved_test_ids populated
```

#### ☐ Verify Event Hub Message
```bash
# In Azure Portal:
[ ] Navigate to Event Hub → Metrics
[ ] Check "Incoming Messages" metric
[ ] Verify message count increased
```

#### ☐ Verify Database Records
```bash
# Connect to database
[ ] docker exec -it coderunner-postgres psql -U postgres -d code_runner_db

# Check executions
[ ] SELECT id, solution_id, success, created_at FROM executions ORDER BY created_at DESC LIMIT 5;

# Should see recent test execution
```

## 🔒 Security Hardening

### Pre-Production Security

#### ☐ Secrets Management
- [ ] Moved secrets to Azure Key Vault
- [ ] Removed hardcoded credentials from .env
- [ ] Configured Managed Identity (if using Azure)
- [ ] Set up secret rotation policy

#### ☐ Network Security
- [ ] Configured Azure NSG (Network Security Groups)
- [ ] Enabled Private Endpoints for Event Hub (production)
- [ ] Configured firewall rules for PostgreSQL
- [ ] Disabled public access where possible
- [ ] Set up VNet integration

#### ☐ Database Security
- [ ] Changed default PostgreSQL password
- [ ] Created application-specific database user
- [ ] Enabled SSL/TLS (DB_SSLMODE=require)
- [ ] Configured connection encryption
- [ ] Set up automated backups

#### ☐ Event Hub Security
- [ ] Created custom SAS policy with minimal permissions
- [ ] Removed Manage permission from application policy
- [ ] Configured IP filtering rules
- [ ] Enabled threat detection
- [ ] Set up audit logging

#### ☐ Container Security
- [ ] Running as non-root user where possible
- [ ] Implemented resource limits
- [ ] Regular image updates scheduled
- [ ] Vulnerability scanning enabled
- [ ] Secret scanning in CI/CD

## 📊 Monitoring Setup

### Logging

#### ☐ Application Logs
```bash
# Verify logging configuration
[ ] LOG_LEVEL=info (use debug only for troubleshooting)
[ ] LOG_FORMAT=json (for structured logging)

# Test log output
[ ] docker-compose logs -f coderunner
```

#### ☐ Centralized Logging (Optional)
- [ ] Azure Application Insights configured
- [ ] Log Analytics workspace created
- [ ] Container Insights enabled
- [ ] Log retention policy set

### Metrics

#### ☐ Azure Monitor
- [ ] Application Insights instrumentation key configured
- [ ] Custom metrics tracked
- [ ] Performance counters enabled
- [ ] Dependency tracking enabled

#### ☐ Container Metrics
```bash
# Monitor resource usage
[ ] docker stats coderunner-service

# Should show CPU, memory, network usage
```

### Alerts

#### ☐ Critical Alerts
- [ ] High error rate (>5% in 5 minutes)
- [ ] Service down (health check fails)
- [ ] Database connection failures
- [ ] Event Hub connection errors
- [ ] High memory usage (>80%)
- [ ] High CPU usage (>80%)

#### ☐ Warning Alerts
- [ ] Elevated response times (>2s)
- [ ] Message processing delays
- [ ] Low disk space
- [ ] Approaching throughput limits

## 🚀 Production Deployment

### Pre-Production Checklist

- [ ] All tests passing in staging environment
- [ ] Load testing completed
- [ ] Performance benchmarks met
- [ ] Security scan passed
- [ ] Documentation updated
- [ ] Runbook prepared
- [ ] Rollback plan documented

### Deployment Steps

#### ☐ Phase 1: Infrastructure
```bash
# 1. Create production resource group
[ ] az group create --name coderunner-prod-rg --location eastus

# 2. Create Event Hub namespace
[ ] az eventhubs namespace create --name prod-eventhub-ns --resource-group coderunner-prod-rg --sku Standard

# 3. Create Event Hub
[ ] az eventhubs eventhub create --name challenge-completed --namespace-name prod-eventhub-ns --resource-group coderunner-prod-rg

# 4. Create PostgreSQL server
[ ] az postgres flexible-server create --name prod-postgres --resource-group coderunner-prod-rg
```

#### ☐ Phase 2: Configuration
```bash
# Update production .env
[ ] cp .env.example .env.production
[ ] Edit .env.production with production values

# Validate production config
[ ] ./scripts/validate-config.sh
```

#### ☐ Phase 3: Build and Push
```bash
# Build production image
[ ] docker build -t coderunner-service:v1.0.0 .

# Tag for Azure Container Registry
[ ] docker tag coderunner-service:v1.0.0 myregistry.azurecr.io/coderunner-service:v1.0.0

# Login to ACR
[ ] az acr login --name myregistry

# Push image
[ ] docker push myregistry.azurecr.io/coderunner-service:v1.0.0
```

#### ☐ Phase 4: Deploy
```bash
# Option A: Azure Container Instances
[ ] az container create \
    --resource-group coderunner-prod-rg \
    --name coderunner-aci \
    --image myregistry.azurecr.io/coderunner-service:v1.0.0 \
    --cpu 2 --memory 4 \
    --environment-variables @env-vars.txt \
    --ports 9084 8084

# Option B: Azure Kubernetes Service
[ ] kubectl apply -f k8s/deployment.yaml
[ ] kubectl apply -f k8s/service.yaml
```

#### ☐ Phase 5: Verify Production
```bash
# Check health
[ ] curl https://prod-coderunner.azure.com/health

# Run smoke tests
[ ] grpcurl prod-coderunner.azure.com:9084 \
    com.levelupjourney.coderunner.CodeExecutionService/HealthCheck

# Monitor initial traffic
[ ] Watch Azure Monitor for 15-30 minutes
```

### Post-Deployment Verification

#### ☐ Functional Verification
- [ ] Health endpoints responding
- [ ] gRPC service accepting requests
- [ ] Database connections stable
- [ ] Event Hub messages flowing
- [ ] Service discovery registration (if enabled)

#### ☐ Performance Verification
- [ ] Response times within SLA (<1s average)
- [ ] Throughput meets requirements
- [ ] CPU usage normal (<50% average)
- [ ] Memory usage stable (<60% average)
- [ ] No error spikes

#### ☐ Integration Verification
- [ ] Events published to Event Hub
- [ ] Database records created
- [ ] Service discoverable (if using Eureka)
- [ ] Dependent services can connect

## 📝 Post-Deployment Tasks

### Documentation

- [ ] Updated deployment runbook
- [ ] Documented configuration changes
- [ ] Updated architecture diagrams
- [ ] Created troubleshooting guide
- [ ] Updated API documentation

### Communication

- [ ] Notified stakeholders of deployment
- [ ] Updated status page
- [ ] Sent deployment summary email
- [ ] Scheduled post-deployment review

### Monitoring

- [ ] Set up 24-hour monitoring watch
- [ ] Reviewed initial metrics
- [ ] Verified alerts are working
- [ ] Checked error logs

## 🔄 Rollback Plan

### Criteria for Rollback

Roll back immediately if:
- [ ] Health checks fail for >5 minutes
- [ ] Error rate >10%
- [ ] Database corruption detected
- [ ] Critical security vulnerability discovered

### Rollback Steps

```bash
# 1. Stop current version
[ ] docker-compose down

# 2. Revert to previous image
[ ] docker pull myregistry.azurecr.io/coderunner-service:v0.9.0

# 3. Update configuration
[ ] cp .env.v0.9.0 .env

# 4. Start previous version
[ ] docker-compose up -d

# 5. Verify rollback
[ ] curl http://localhost:8084/health
[ ] Run smoke tests
```

## 📞 Support Information

### On-Call Contacts

- **Primary**: ___________________________
- **Secondary**: ___________________________
- **Manager**: ___________________________

### Key Resources

- **Azure Portal**: https://portal.azure.com
- **Monitoring Dashboard**: ___________________________
- **Log Analytics**: ___________________________
- **Runbook**: ___________________________

### Emergency Procedures

1. **Service Down**: Check health endpoint, review logs, restart service
2. **Database Issues**: Check connection, verify firewall, review queries
3. **Event Hub Issues**: Verify connection string, check network, review SAS policy
4. **High Load**: Check metrics, scale resources, enable throttling

## ✅ Deployment Sign-Off

### Deployment Information

- **Deployment Date**: _______________
- **Deployed By**: _______________
- **Version**: _______________
- **Environment**: ☐ Development  ☐ Staging  ☐ Production

### Sign-Off

- [ ] Technical Lead approval
- [ ] Security review completed
- [ ] Operations team notified
- [ ] Documentation updated
- [ ] Monitoring configured
- [ ] Backup verified

**Deployment Status**: ☐ Success  ☐ Partial  ☐ Failed  ☐ Rolled Back

**Notes**:
_________________________________________________________________
_________________________________________________________________
_________________________________________________________________

---

**Completed**: _______________  
**Signed**: _______________
