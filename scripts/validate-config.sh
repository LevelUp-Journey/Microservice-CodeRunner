#!/bin/bash

# ============================================================================
# Configuration Validation Script for CodeRunner Microservice
# ============================================================================
# This script validates all required environment variables and connections
# before deploying the Docker containers.
# ============================================================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color
BOLD='\033[1m'

# Counters
ERRORS=0
WARNINGS=0
PASSED=0

# ============================================================================
# Helper Functions
# ============================================================================

print_header() {
    echo ""
    echo -e "${BOLD}${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}${BLUE}  $1${NC}"
    echo -e "${BOLD}${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
    ((PASSED++))
}

print_error() {
    echo -e "${RED}✗${NC} $1"
    ((ERRORS++))
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
    ((WARNINGS++))
}

print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

check_variable() {
    local var_name=$1
    local var_value=$2
    local is_required=$3
    local description=$4

    if [ -z "$var_value" ]; then
        if [ "$is_required" = "true" ]; then
            print_error "$var_name is not set - $description"
            return 1
        else
            print_warning "$var_name is not set (optional) - $description"
            return 0
        fi
    else
        print_success "$var_name is set"
        return 0
    fi
}

check_connection() {
    local host=$1
    local port=$2
    local name=$3

    if timeout 3 bash -c "echo > /dev/tcp/$host/$port" 2>/dev/null; then
        print_success "Connection to $name ($host:$port) successful"
        return 0
    else
        print_error "Cannot connect to $name ($host:$port)"
        return 1
    fi
}

# ============================================================================
# Load Environment Variables
# ============================================================================

print_header "Loading Environment Configuration"

if [ ! -f .env ]; then
    print_error ".env file not found!"
    echo ""
    print_info "Run: cp .env.example .env"
    print_info "Then edit .env with your configuration"
    exit 1
fi

print_success ".env file found"

# Load .env file
set -a
source .env
set +a

print_success "Environment variables loaded"

# ============================================================================
# Validate Application Configuration
# ============================================================================

print_header "Validating Application Configuration"

check_variable "APP_NAME" "$APP_NAME" false "Application name"
check_variable "API_VERSION" "$API_VERSION" false "API version"
check_variable "PORT" "$PORT" true "HTTP port for health checks"
check_variable "GRPC_PORT" "$GRPC_PORT" true "gRPC server port"

# Validate port numbers
if [ -n "$PORT" ]; then
    if ! [[ "$PORT" =~ ^[0-9]+$ ]] || [ "$PORT" -lt 1024 ] || [ "$PORT" -gt 65535 ]; then
        print_error "PORT must be a valid port number (1024-65535)"
    else
        print_success "PORT is valid ($PORT)"
    fi
fi

if [ -n "$GRPC_PORT" ]; then
    if ! [[ "$GRPC_PORT" =~ ^[0-9]+$ ]] || [ "$GRPC_PORT" -lt 1024 ] || [ "$GRPC_PORT" -gt 65535 ]; then
        print_error "GRPC_PORT must be a valid port number (1024-65535)"
    else
        print_success "GRPC_PORT is valid ($GRPC_PORT)"
    fi
fi

# ============================================================================
# Validate Database Configuration
# ============================================================================

print_header "Validating Database Configuration"

check_variable "DB_HOST" "$DB_HOST" true "Database hostname"
check_variable "DB_PORT" "$DB_PORT" true "Database port"
check_variable "DB_USER" "$DB_USER" true "Database username"
check_variable "DB_PASSWORD" "$DB_PASSWORD" true "Database password"
check_variable "DB_NAME" "$DB_NAME" true "Database name"
check_variable "DB_SSLMODE" "$DB_SSLMODE" false "SSL mode"

# Check if Azure PostgreSQL
if [[ "$DB_HOST" == *".postgres.database.azure.com"* ]]; then
    print_info "Detected Azure Database for PostgreSQL"

    if [ "$DB_SSLMODE" != "require" ]; then
        print_warning "DB_SSLMODE should be 'require' for Azure PostgreSQL"
    fi

    if [[ "$DB_USER" != *"@"* ]]; then
        print_warning "Azure PostgreSQL user should include @servername (e.g., admin@myserver)"
    fi
fi

# Validate connection pool settings
if [ -n "$DB_MAX_OPEN_CONNS" ]; then
    if ! [[ "$DB_MAX_OPEN_CONNS" =~ ^[0-9]+$ ]]; then
        print_error "DB_MAX_OPEN_CONNS must be a number"
    else
        print_success "DB_MAX_OPEN_CONNS is valid ($DB_MAX_OPEN_CONNS)"
    fi
fi

# ============================================================================
# Validate Kafka / Azure Event Hub Configuration
# ============================================================================

print_header "Validating Kafka / Azure Event Hub Configuration"

check_variable "KAFKA_BOOTSTRAP_SERVERS" "$KAFKA_BOOTSTRAP_SERVERS" true "Kafka bootstrap servers"
check_variable "KAFKA_CONNECTION_STRING" "$KAFKA_CONNECTION_STRING" true "Kafka/Event Hub connection string"
check_variable "KAFKA_TOPIC" "$KAFKA_TOPIC" true "Kafka topic/Event Hub name"
check_variable "KAFKA_CONSUMER_GROUP" "$KAFKA_CONSUMER_GROUP" false "Consumer group ID"

# Validate Azure Event Hub format
if [ -n "$KAFKA_BOOTSTRAP_SERVERS" ]; then
    if [[ "$KAFKA_BOOTSTRAP_SERVERS" == *".servicebus.windows.net:9093"* ]]; then
        print_success "Azure Event Hub format detected"

        # Check if connection string is for Event Hub
        if [[ "$KAFKA_CONNECTION_STRING" != *"Endpoint=sb://"* ]]; then
            print_error "KAFKA_CONNECTION_STRING must start with 'Endpoint=sb://' for Azure Event Hub"
        else
            print_success "Connection string format is valid"
        fi

        # Check if EntityPath is present
        if [[ "$KAFKA_CONNECTION_STRING" != *"EntityPath="* ]]; then
            print_error "KAFKA_CONNECTION_STRING must include 'EntityPath=' parameter"
        else
            # Extract EntityPath
            ENTITY_PATH=$(echo "$KAFKA_CONNECTION_STRING" | grep -oP 'EntityPath=\K[^;]+')

            if [ "$ENTITY_PATH" != "$KAFKA_TOPIC" ]; then
                print_warning "EntityPath ($ENTITY_PATH) doesn't match KAFKA_TOPIC ($KAFKA_TOPIC)"
                print_info "These should match for Azure Event Hub"
            else
                print_success "EntityPath matches KAFKA_TOPIC"
            fi
        fi

        # Check SharedAccessKey presence
        if [[ "$KAFKA_CONNECTION_STRING" != *"SharedAccessKey="* ]]; then
            print_error "KAFKA_CONNECTION_STRING must include 'SharedAccessKey=' parameter"
        fi

        # Check SharedAccessKeyName presence
        if [[ "$KAFKA_CONNECTION_STRING" != *"SharedAccessKeyName="* ]]; then
            print_error "KAFKA_CONNECTION_STRING must include 'SharedAccessKeyName=' parameter"
        fi

    elif [[ "$KAFKA_BOOTSTRAP_SERVERS" == *":9092"* ]]; then
        print_info "Standard Kafka format detected"
    else
        print_warning "Unusual Kafka bootstrap server format"
    fi
fi

# Validate SASL settings
check_variable "KAFKA_SASL_MECHANISM" "$KAFKA_SASL_MECHANISM" false "SASL mechanism"
check_variable "KAFKA_SECURITY_PROTOCOL" "$KAFKA_SECURITY_PROTOCOL" false "Security protocol"

if [[ "$KAFKA_BOOTSTRAP_SERVERS" == *".servicebus.windows.net"* ]]; then
    if [ "$KAFKA_SASL_MECHANISM" != "PLAIN" ]; then
        print_error "KAFKA_SASL_MECHANISM must be 'PLAIN' for Azure Event Hub"
    fi

    if [ "$KAFKA_SECURITY_PROTOCOL" != "SASL_SSL" ]; then
        print_error "KAFKA_SECURITY_PROTOCOL must be 'SASL_SSL' for Azure Event Hub"
    fi
fi

# ============================================================================
# Validate Service Discovery Configuration
# ============================================================================

print_header "Validating Service Discovery Configuration"

check_variable "SERVICE_DISCOVERY_ENABLED" "$SERVICE_DISCOVERY_ENABLED" false "Service discovery enable flag"
check_variable "SERVICE_DISCOVERY_URL" "$SERVICE_DISCOVERY_URL" false "Eureka server URL"
check_variable "SERVICE_NAME" "$SERVICE_NAME" false "Service name"
check_variable "SERVICE_PUBLIC_IP" "$SERVICE_PUBLIC_IP" false "Public IP address"

if [ "$SERVICE_DISCOVERY_ENABLED" = "true" ]; then
    if [ -z "$SERVICE_DISCOVERY_URL" ]; then
        print_error "SERVICE_DISCOVERY_URL is required when SERVICE_DISCOVERY_ENABLED=true"
    fi

    if [ -z "$SERVICE_NAME" ]; then
        print_warning "SERVICE_NAME is not set, using default"
    fi
fi

# ============================================================================
# Validate Logging Configuration
# ============================================================================

print_header "Validating Logging Configuration"

check_variable "LOG_LEVEL" "$LOG_LEVEL" false "Log level"
check_variable "LOG_FORMAT" "$LOG_FORMAT" false "Log format"

if [ -n "$LOG_LEVEL" ]; then
    case "$LOG_LEVEL" in
        debug|info|warn|error)
            print_success "LOG_LEVEL is valid ($LOG_LEVEL)"
            ;;
        *)
            print_error "LOG_LEVEL must be one of: debug, info, warn, error"
            ;;
    esac
fi

if [ -n "$LOG_FORMAT" ]; then
    case "$LOG_FORMAT" in
        json|text)
            print_success "LOG_FORMAT is valid ($LOG_FORMAT)"
            ;;
        *)
            print_error "LOG_FORMAT must be one of: json, text"
            ;;
    esac
fi

# ============================================================================
# Test Network Connectivity (Optional)
# ============================================================================

print_header "Testing Network Connectivity (Optional)"

print_info "Attempting to test connections... (may fail in restricted networks)"

# Test DNS resolution for Azure Event Hub
if [ -n "$KAFKA_BOOTSTRAP_SERVERS" ]; then
    EVENT_HUB_HOST=$(echo "$KAFKA_BOOTSTRAP_SERVERS" | cut -d: -f1)

    if command -v nslookup &> /dev/null; then
        if nslookup "$EVENT_HUB_HOST" &> /dev/null; then
            print_success "DNS resolution for $EVENT_HUB_HOST works"
        else
            print_warning "Cannot resolve $EVENT_HUB_HOST (check DNS or network)"
        fi
    else
        print_info "nslookup not available, skipping DNS test"
    fi
fi

# Test database connection (only if using localhost or accessible host)
if [ "$DB_HOST" = "localhost" ] || [ "$DB_HOST" = "postgres" ]; then
    print_info "Database appears to be local, skipping connection test"
elif [[ "$DB_HOST" == *".postgres.database.azure.com"* ]]; then
    print_info "Database is Azure PostgreSQL, connection will be tested at runtime"
fi

# ============================================================================
# Check Docker Installation
# ============================================================================

print_header "Checking Docker Installation"

if command -v docker &> /dev/null; then
    print_success "Docker is installed ($(docker --version))"

    # Check if Docker daemon is running
    if docker info &> /dev/null; then
        print_success "Docker daemon is running"
    else
        print_error "Docker daemon is not running"
    fi
else
    print_error "Docker is not installed"
fi

if command -v docker-compose &> /dev/null; then
    print_success "Docker Compose is installed ($(docker-compose --version))"
else
    print_error "Docker Compose is not installed"
fi

# ============================================================================
# Summary
# ============================================================================

print_header "Validation Summary"

echo ""
echo -e "${BOLD}Results:${NC}"
echo -e "  ${GREEN}✓ Passed:${NC}   $PASSED"
echo -e "  ${YELLOW}⚠ Warnings:${NC} $WARNINGS"
echo -e "  ${RED}✗ Errors:${NC}   $ERRORS"
echo ""

if [ $ERRORS -eq 0 ] && [ $WARNINGS -eq 0 ]; then
    echo -e "${GREEN}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}${BOLD}  ✓ Configuration is valid! Ready to deploy.${NC}"
    echo -e "${GREEN}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo -e "${BLUE}Next steps:${NC}"
    echo -e "  ${BLUE}1.${NC} Run: ${BOLD}make build${NC} or ${BOLD}docker-compose build${NC}"
    echo -e "  ${BLUE}2.${NC} Run: ${BOLD}make up${NC} or ${BOLD}docker-compose up -d${NC}"
    echo -e "  ${BLUE}3.${NC} Check: ${BOLD}make health${NC} or ${BOLD}curl http://localhost:8084/health${NC}"
    echo ""
    exit 0
elif [ $ERRORS -eq 0 ]; then
    echo -e "${YELLOW}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${YELLOW}${BOLD}  ⚠ Configuration has warnings but is usable.${NC}"
    echo -e "${YELLOW}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo -e "${BLUE}You can proceed with deployment, but consider fixing warnings.${NC}"
    echo ""
    exit 0
else
    echo -e "${RED}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${RED}${BOLD}  ✗ Configuration has errors! Please fix before deploying.${NC}"
    echo -e "${RED}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo -e "${BLUE}Please review the errors above and update your .env file.${NC}"
    echo ""
    exit 1
fi
