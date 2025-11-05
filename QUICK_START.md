# 🚀 Quick Start Guide - CodeRunner Microservice

Get your CodeRunner gRPC microservice up and running in minutes with Docker and Azure Event Hub.

## ⚡ Prerequisites

- **Docker** 20.10+ with Docker Compose
- **Azure Account** with Event Hub namespace
- **5 minutes** of your time

## 🎯 Quick Setup (3 Steps)

### Step 1: Clone and Configure

```bash
# Clone repository
git clone <repository-url>
cd Microservice-CodeRunner

# Create environment file
cp .env.example .env

# Edit with your Azure credentials
nano .env  # or use your favorite editor
```

### Step 2: Configure Azure Event Hub

**Get your Event Hub connection details:**

1. Go to [Azure Portal](https://portal.azure.com)
2. Navigate to: **Event Hubs** → Your Namespace → **Shared access policies**
3. Click **RootManageSharedAccessKey**
4. Copy the **Connection string–primary key**

**Update your `.env` file:**

```bash
# Required: Azure Event Hub Configuration
KAFKA_BOOTSTRAP_SERVERS=your-namespace.servicebus.windows.net:9093
KAFKA_CONNECTION_STRING=Endpoint=sb://your-namespace.servicebus.windows.net/;SharedAccessKeyName=RootManageSharedAccessKey;SharedAccessKey=your-key==;EntityPath=challenge-completed
KAFKA_TOPIC=challenge-completed

# Optional: Database (defaults to local PostgreSQL)
DB_HOST=postgres
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=code_runner_db
```

### Step 3: Deploy

```bash
# Option A: Using Make (recommended)
make validate  # Validate configuration
make deploy    # Build and start services

# Option B: Using Docker Compose directly
docker-compose up -d --build

# View logs
make logs
# or
docker-compose logs -f coderunner
```

## ✅ Verify Deployment

```bash
# Check service health
curl http://localhost:8084/health

# Should return: {"status":"healthy"}
```

## 🧪 Test the Service

```bash
# Install grpcurl (if not installed)
# macOS: brew install grpcurl
# Linux: apt-get install grpcurl or snap install grpcurl

# Test gRPC endpoint
grpcurl -plaintext -d '{
  "solution_id": "test_001",
  "challenge_id": "factorial",
  "student_id": "test_student",
  "code": "function factorial(n) { return n <= 1 ? 1 : n * factorial(n-1); }",
  "language": "javascript"
}' localhost:9084 com.levelupjourney.coderunner.CodeExecutionService/ExecuteCode
```

**Expected Response:**
```json
{
  "success": true,
  "message": "All 2 tests passed",
  "execution_id": "exec_...",
  "approved_test_ids": ["test_1", "test_2"]
}
```

## 🎛️ Access Management Tools

### pgAdmin (Database Management)

- **URL**: http://localhost:5050
- **Email**: admin@coderunner.local
- **Password**: admin123

### Docker Stats

```bash
# View resource usage
make stats

# View running containers
make status

# View Docker containers inside DinD
docker exec -it coderunner-service docker ps
```

## 🔧 Common Commands

```bash
# Start services
make up

# Stop services
make down

# Restart services
make restart

# View logs
make logs

# Run validation
make validate

# Health check
make health

# Access shell
make shell

# Database shell
make db-shell

# Clean up
make clean
```

## 📊 Monitoring

### View Logs

```bash
# All services
docker-compose logs -f

# CodeRunner only
docker-compose logs -f coderunner

# PostgreSQL only
docker-compose logs -f postgres

# Last 100 lines
docker-compose logs --tail=100
```

### Check Azure Event Hub

1. Go to Azure Portal → Your Event Hub Namespace
2. Click **Metrics**
3. View **Incoming Messages** and **Outgoing Messages**

## 🐛 Troubleshooting

### Issue: Docker daemon not starting

```bash
# Check if privileged mode is enabled
docker inspect coderunner-service | grep Privileged

# Should show: "Privileged": true
```

### Issue: Cannot connect to Event Hub

```bash
# Validate configuration
./scripts/validate-config.sh

# Test SSL connectivity
openssl s_client -connect your-namespace.servicebus.windows.net:9093
```

### Issue: Database connection failed

```bash
# Check PostgreSQL status
docker-compose ps postgres

# View database logs
docker-compose logs postgres

# Connect to database manually
docker exec -it coderunner-postgres psql -U postgres -d code_runner_db
```

### Issue: Port already in use

```bash
# Change ports in .env file
PORT=8085
GRPC_PORT=9085

# Restart services
docker-compose down
docker-compose up -d
```

## 🔒 Security Checklist

Before going to production:

- [ ] Change default database password
- [ ] Use Azure Key Vault for secrets
- [ ] Enable Private Endpoints for Event Hub
- [ ] Configure firewall rules
- [ ] Use SSL for database (DB_SSLMODE=require)
- [ ] Rotate access keys regularly
- [ ] Set up monitoring and alerts
- [ ] Review network security groups

## 📚 Next Steps

1. **Read Full Documentation**
   - [Docker Setup Guide](DOCKER_SETUP.md)
   - [Azure Event Hub Setup](docs/AZURE_EVENT_HUB_SETUP.md)
   - [Main README](README.md)

2. **Configure Production Settings**
   - Set up Azure Key Vault
   - Configure monitoring
   - Set up CI/CD pipeline

3. **Scale Your Service**
   - Configure auto-scaling
   - Set up load balancing
   - Add replica sets

## 🆘 Need Help?

### Documentation
- [Docker Setup Guide](DOCKER_SETUP.md) - Complete Docker configuration
- [Azure Event Hub Guide](docs/AZURE_EVENT_HUB_SETUP.md) - Kafka/SASL setup
- [Makefile Reference](Makefile) - All available commands

### Common Issues
```bash
# Run diagnostics
make troubleshoot

# Validate configuration
./scripts/validate-config.sh

# Check system status
make info
```

### Support
- Check logs: `make logs`
- Review configuration: `cat .env`
- Test connections: `make health`
- Open an issue on GitHub

## 📝 Configuration Templates

### Local Development

```bash
# .env for local development
KAFKA_BOOTSTRAP_SERVERS=localhost:9092
KAFKA_CONNECTION_STRING=
KAFKA_SECURITY_PROTOCOL=PLAINTEXT
DB_HOST=postgres
DB_SSLMODE=disable
SERVICE_DISCOVERY_ENABLED=false
```

### Azure Production

```bash
# .env for Azure production
KAFKA_BOOTSTRAP_SERVERS=prod-ns.servicebus.windows.net:9093
KAFKA_CONNECTION_STRING=Endpoint=sb://prod-ns.servicebus.windows.net/;SharedAccessKeyName=...;SharedAccessKey=...;EntityPath=...
KAFKA_SECURITY_PROTOCOL=SASL_SSL
KAFKA_SASL_MECHANISM=PLAIN
DB_HOST=prod-server.postgres.database.azure.com
DB_SSLMODE=require
SERVICE_DISCOVERY_ENABLED=true
SERVICE_DISCOVERY_URL=http://eureka:8761/eureka
```

## 🎉 Success!

You now have a running CodeRunner microservice with:

- ✅ gRPC server on port 9084
- ✅ Health check API on port 8084
- ✅ PostgreSQL database
- ✅ Azure Event Hub integration
- ✅ Docker-in-Docker for code execution
- ✅ pgAdmin for database management

**Happy Coding! 🚀**

---

**Last Updated**: 2025
**Version**: 1.0.0