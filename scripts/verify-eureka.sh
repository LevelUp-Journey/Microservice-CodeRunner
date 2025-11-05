#!/bin/bash

# Eureka Service Discovery Verification Script
# This script verifies that the Code Runner service is properly registered with Eureka

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
EUREKA_URL="${SERVICE_DISCOVERY_URL:-http://localhost:8761}"
SERVICE_NAME="${SERVICE_NAME:-CODE-RUNNER-SERVICE}"
CONTAINER_NAME="${1:-code-runner-service}"
MAX_RETRIES=30
RETRY_DELAY=2

echo -e "${BLUE}╔════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  Eureka Service Discovery Verification        ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════╝${NC}"
echo ""

# Function to print status
print_status() {
    local status=$1
    local message=$2
    if [ "$status" == "OK" ]; then
        echo -e "${GREEN}✅ ${message}${NC}"
    elif [ "$status" == "ERROR" ]; then
        echo -e "${RED}❌ ${message}${NC}"
    elif [ "$status" == "WARN" ]; then
        echo -e "${YELLOW}⚠️  ${message}${NC}"
    else
        echo -e "${BLUE}ℹ️  ${message}${NC}"
    fi
}

# Function to check if container is running
check_container() {
    print_status "INFO" "Checking if container '$CONTAINER_NAME' is running..."

    if docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        print_status "OK" "Container is running"
        return 0
    else
        print_status "ERROR" "Container is not running"
        echo ""
        echo "Please start the container first:"
        echo "  docker-compose up -d $CONTAINER_NAME"
        exit 1
    fi
}

# Function to check container logs
check_logs() {
    print_status "INFO" "Checking container logs for registration messages..."

    if docker logs "$CONTAINER_NAME" 2>&1 | grep -q "Service registered in Eureka"; then
        print_status "OK" "Registration message found in logs"
        return 0
    else
        print_status "WARN" "Registration message not yet found in logs"
        return 1
    fi
}

# Function to check Eureka server
check_eureka_server() {
    print_status "INFO" "Checking Eureka server at: $EUREKA_URL"

    local eureka_base_url=$(echo "$EUREKA_URL" | sed 's|/eureka.*||')

    if curl -s -f "$eureka_base_url" > /dev/null 2>&1; then
        print_status "OK" "Eureka server is reachable"
        return 0
    else
        print_status "ERROR" "Cannot reach Eureka server at $eureka_base_url"
        echo ""
        echo "Please check:"
        echo "  1. Eureka server is running"
        echo "  2. SERVICE_DISCOVERY_URL is correct"
        echo "  3. Services are on the same Docker network"
        exit 1
    fi
}

# Function to check service registration in Eureka
check_eureka_registration() {
    print_status "INFO" "Checking if service is registered in Eureka..."

    local eureka_base_url=$(echo "$EUREKA_URL" | sed 's|/eureka.*||')
    local apps_url="${eureka_base_url}/eureka/apps"

    local retry=0
    while [ $retry -lt $MAX_RETRIES ]; do
        if curl -s -f "$apps_url" > /dev/null 2>&1; then
            local response=$(curl -s -H "Accept: application/json" "$apps_url/$SERVICE_NAME")

            if echo "$response" | grep -q "$SERVICE_NAME"; then
                print_status "OK" "Service is registered in Eureka"

                # Extract instance information
                local instance_id=$(echo "$response" | grep -o '"instanceId":"[^"]*"' | head -1 | cut -d'"' -f4)
                local status=$(echo "$response" | grep -o '"status":"[^"]*"' | head -1 | cut -d'"' -f4)
                local hostname=$(echo "$response" | grep -o '"hostName":"[^"]*"' | head -1 | cut -d'"' -f4)

                echo ""
                echo -e "${BLUE}📋 Registration Details:${NC}"
                echo "   Instance ID: $instance_id"
                echo "   Status: $status"
                echo "   Hostname: $hostname"

                if [ "$status" == "UP" ]; then
                    print_status "OK" "Service status is UP"
                    return 0
                else
                    print_status "WARN" "Service status is $status (not UP)"
                    return 1
                fi
            fi
        fi

        retry=$((retry + 1))
        if [ $retry -lt $MAX_RETRIES ]; then
            echo -e "${YELLOW}⏳ Waiting for registration... (attempt $retry/$MAX_RETRIES)${NC}"
            sleep $RETRY_DELAY
        fi
    done

    print_status "ERROR" "Service not found in Eureka after $MAX_RETRIES attempts"
    return 1
}

# Function to check heartbeat in logs
check_heartbeat() {
    print_status "INFO" "Checking for heartbeat messages..."

    if docker logs "$CONTAINER_NAME" --tail 50 2>&1 | grep -q "Heartbeat sent successfully"; then
        print_status "OK" "Heartbeat is being sent successfully"

        # Show last heartbeat
        local last_heartbeat=$(docker logs "$CONTAINER_NAME" --tail 100 2>&1 | grep "Heartbeat sent successfully" | tail -1)
        echo "   Last: $last_heartbeat"
        return 0
    else
        print_status "WARN" "No recent heartbeat messages found"
        return 1
    fi
}

# Function to test connectivity from another service
check_connectivity() {
    print_status "INFO" "Testing service connectivity..."

    # Try to find another container on the same network
    local network=$(docker inspect "$CONTAINER_NAME" --format '{{range $net, $v := .NetworkSettings.Networks}}{{$net}}{{end}}' | head -1)

    if [ -z "$network" ]; then
        print_status "WARN" "Cannot determine container network"
        return 1
    fi

    print_status "INFO" "Container is on network: $network"

    # Check if service is accessible via hostname
    if docker run --rm --network "$network" curlimages/curl:latest curl -s -f "http://$CONTAINER_NAME:8084/health" > /dev/null 2>&1; then
        print_status "OK" "Service is accessible via hostname ($CONTAINER_NAME:8084)"
        return 0
    else
        print_status "WARN" "Service may not be accessible via hostname"
        return 1
    fi
}

# Function to display configuration
display_config() {
    echo ""
    echo -e "${BLUE}📝 Current Configuration:${NC}"
    echo "   Container Name: $CONTAINER_NAME"
    echo "   Service Name: $SERVICE_NAME"
    echo "   Eureka URL: $EUREKA_URL"

    # Get environment variables from container
    local env_vars=$(docker inspect "$CONTAINER_NAME" --format '{{range .Config.Env}}{{println .}}{{end}}')

    if echo "$env_vars" | grep -q "HOSTNAME="; then
        local hostname=$(echo "$env_vars" | grep "HOSTNAME=" | cut -d'=' -f2)
        echo "   Hostname: $hostname"
    fi

    if echo "$env_vars" | grep -q "GRPC_PORT="; then
        local grpc_port=$(echo "$env_vars" | grep "GRPC_PORT=" | cut -d'=' -f2)
        echo "   gRPC Port: $grpc_port"
    fi

    echo ""
}

# Main execution
main() {
    echo ""

    # Run checks
    check_container
    echo ""

    display_config

    check_eureka_server
    echo ""

    check_logs
    echo ""

    if check_eureka_registration; then
        echo ""
        check_heartbeat
        echo ""
        check_connectivity
        echo ""

        echo -e "${GREEN}╔════════════════════════════════════════════════╗${NC}"
        echo -e "${GREEN}║  ✅ All checks passed!                         ║${NC}"
        echo -e "${GREEN}║  Service is properly registered with Eureka   ║${NC}"
        echo -e "${GREEN}╚════════════════════════════════════════════════╝${NC}"
        echo ""
        echo "🌐 View Eureka Dashboard: $(echo "$EUREKA_URL" | sed 's|/eureka.*||')"
        echo ""
        exit 0
    else
        echo ""
        echo -e "${RED}╔════════════════════════════════════════════════╗${NC}"
        echo -e "${RED}║  ❌ Service registration failed                ║${NC}"
        echo -e "${RED}╚════════════════════════════════════════════════╝${NC}"
        echo ""
        echo "Troubleshooting tips:"
        echo ""
        echo "1. Check container logs:"
        echo "   docker logs $CONTAINER_NAME"
        echo ""
        echo "2. Verify environment variables:"
        echo "   docker exec $CONTAINER_NAME env | grep SERVICE"
        echo ""
        echo "3. Check network connectivity:"
        echo "   docker network inspect <network-name>"
        echo ""
        echo "4. Restart the service:"
        echo "   docker-compose restart $CONTAINER_NAME"
        echo ""
        echo "5. See detailed guide:"
        echo "   cat EUREKA_INTEGRATION.md"
        echo ""
        exit 1
    fi
}

# Handle script arguments
if [ "$1" == "--help" ] || [ "$1" == "-h" ]; then
    echo "Usage: $0 [CONTAINER_NAME]"
    echo ""
    echo "Verifies that the Code Runner service is properly registered with Eureka."
    echo ""
    echo "Arguments:"
    echo "  CONTAINER_NAME    Name of the container to check (default: code-runner-service)"
    echo ""
    echo "Environment variables:"
    echo "  SERVICE_DISCOVERY_URL    Eureka server URL (default: http://localhost:8761)"
    echo "  SERVICE_NAME             Service name in Eureka (default: CODE-RUNNER-SERVICE)"
    echo ""
    echo "Examples:"
    echo "  $0"
    echo "  $0 my-code-runner"
    echo "  SERVICE_DISCOVERY_URL=http://eureka:8761 $0"
    exit 0
fi

# Run main function
main
