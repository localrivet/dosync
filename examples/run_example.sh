#!/bin/bash
# Script to run the Docker Compose replica detection example
set -e

# Colorized output helpers
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &>/dev/null && pwd)"
cd "$SCRIPT_DIR"

# Check if Docker is running
if ! docker info >/dev/null 2>&1; then
    echo -e "${RED}Error: Docker is not running or not accessible${NC}"
    echo "Please start Docker and try again"
    exit 1
fi

# Check if Docker Compose is installed
if ! command -v docker-compose &>/dev/null; then
    echo -e "${RED}Error: docker-compose command not found${NC}"
    echo "Please install Docker Compose and try again"
    exit 1
fi

# Function to clean up resources
function cleanup {
    echo -e "\n${YELLOW}Cleaning up resources...${NC}"
    docker-compose down
}

# Set up trap to catch Ctrl+C and other termination signals
trap cleanup EXIT

# Check if the example binary exists, build it if it doesn't
if [ ! -f "./replica_detector" ]; then
    echo -e "${YELLOW}Building replica detector example...${NC}"
    cd ..
    go build -o examples/replica_detector examples/replica_detection.go
    cd "$SCRIPT_DIR"
fi

# Start Docker Compose services
echo -e "${YELLOW}Starting Docker Compose services...${NC}"
docker-compose up -d

# Wait for containers to fully start
echo -e "${YELLOW}Waiting for containers to start...${NC}"
sleep 3

# Run the replica detector
echo -e "${GREEN}Running replica detector example...${NC}"
./replica_detector

# Ask user if they want to keep the containers running
echo -e "\n${YELLOW}Would you like to keep the Docker containers running? (y/n)${NC}"
read -r answer
if [[ "$answer" =~ ^[Nn] ]]; then
    echo -e "${YELLOW}Stopping Docker Compose services...${NC}"
    docker-compose down
    echo -e "${GREEN}Containers stopped successfully${NC}"
    # Remove the trap since we've already handled cleanup
    trap - EXIT
fi
