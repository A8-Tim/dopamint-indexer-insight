#!/bin/bash

# Dopamint Indexer Setup Script
# This script sets up the development environment for the Dopamint indexer

set -e

echo "======================================"
echo "Dopamint Indexer Setup"
echo "======================================"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if .env exists
if [ ! -f .env ]; then
    echo -e "${YELLOW}Creating .env file from .env.example...${NC}"
    cp .env.example .env
    echo -e "${GREEN}✓ .env file created${NC}"
    echo -e "${YELLOW}⚠ Please edit .env file and add your contract addresses and RPC URL${NC}"
    echo ""
fi

# Check Docker
echo "Checking Docker installation..."
if ! command -v docker &> /dev/null; then
    echo -e "${RED}✗ Docker is not installed${NC}"
    echo "Please install Docker: https://docs.docker.com/get-docker/"
    exit 1
fi
echo -e "${GREEN}✓ Docker is installed${NC}"

# Check Docker Compose
echo "Checking Docker Compose installation..."
if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo -e "${RED}✗ Docker Compose is not installed${NC}"
    echo "Please install Docker Compose: https://docs.docker.com/compose/install/"
    exit 1
fi
echo -e "${GREEN}✓ Docker Compose is installed${NC}"
echo ""

# Check Go installation
echo "Checking Go installation..."
if ! command -v go &> /dev/null; then
    echo -e "${YELLOW}⚠ Go is not installed${NC}"
    echo "Go is required to build the indexer."
    echo "Install from: https://golang.org/doc/install"
else
    GO_VERSION=$(go version | awk '{print $3}')
    echo -e "${GREEN}✓ Go is installed ($GO_VERSION)${NC}"
fi
echo ""

# Create necessary directories
echo "Creating necessary directories..."
mkdir -p logs data/clickhouse data/mongodb data/redis data/kafka
echo -e "${GREEN}✓ Directories created${NC}"
echo ""

# Start infrastructure services
echo "Starting infrastructure services (ClickHouse, MongoDB, Redis, Kafka)..."
docker-compose up -d clickhouse mongodb redis kafka
echo -e "${GREEN}✓ Infrastructure services started${NC}"
echo ""

# Wait for services to be ready
echo "Waiting for services to be ready..."
sleep 10

# Check ClickHouse
echo "Checking ClickHouse..."
until docker-compose exec -T clickhouse clickhouse-client --query "SELECT 1" &> /dev/null; do
    echo "Waiting for ClickHouse..."
    sleep 2
done
echo -e "${GREEN}✓ ClickHouse is ready${NC}"

# Check MongoDB
echo "Checking MongoDB..."
until docker-compose exec -T mongodb mongosh --eval "db.adminCommand('ping')" &> /dev/null; do
    echo "Waiting for MongoDB..."
    sleep 2
done
echo -e "${GREEN}✓ MongoDB is ready${NC}"

# Check Redis
echo "Checking Redis..."
until docker-compose exec -T redis redis-cli ping &> /dev/null; do
    echo "Waiting for Redis..."
    sleep 2
done
echo -e "${GREEN}✓ Redis is ready${NC}"
echo ""

# Initialize databases
echo "Initializing databases..."
./scripts/init-databases.sh
echo ""

echo "======================================"
echo -e "${GREEN}Setup Complete!${NC}"
echo "======================================"
echo ""
echo "Next steps:"
echo "1. Edit .env file and add your Dopamint contract addresses"
echo "2. Configure RPC_URL in .env for Base chain"
echo "3. Run 'docker-compose up -d' to start all services"
echo "4. Run './scripts/start-indexer.sh backfill' to start backfilling"
echo "5. Run './scripts/start-indexer.sh committer' to start live indexing"
echo ""
echo "Monitoring:"
echo "- Grafana: http://localhost:3000 (admin/admin)"
echo "- Prometheus: http://localhost:9090"
echo "- ClickHouse: http://localhost:8123"
echo ""
