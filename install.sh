#!/bin/bash

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Default values
COMPOSE_FILE="docker-compose.yml"
DO_TOKEN=""
BACKUP_DIR="backups"
CHECK_INTERVAL="1m"
VERBOSE="--verbose"

# Banner
echo -e "${GREEN}"
echo "    ____   ____   _____                       "
echo "   / __ \ / __ \ / ___/ ___  __ ____  _____"
echo "  / / / // / / / \__ \// / / // __ \/ ___/"
echo " / /_/ // /_/ / ___/ // /_/ // / / / /__  "
echo "/_____//_____//_____/ \__, //_/ /_/\___/  "
echo "                     /____/                "
echo -e "${NC}"
echo -e "${YELLOW}Container Registry Synchronization Service${NC}"
echo

# Help function
function show_help {
    echo -e "Usage: ${YELLOW}$0 [options]${NC}"
    echo
    echo "This script adds DOSync to your Docker Compose project to auto-update services"
    echo "from various container registries."
    echo
    echo "Supported registries:"
    echo "  • Docker Hub"
    echo "  • GitHub Container Registry (GHCR)"
    echo "  • Google Container Registry (GCR)"
    echo "  • Azure Container Registry (ACR)"
    echo "  • DigitalOcean Container Registry (DOCR)"
    echo "  • Amazon Elastic Container Registry (ECR)"
    echo "  • Harbor"
    echo "  • Quay.io"
    echo "  • Custom Docker-compatible registries"
    echo
    echo "Options:"
    echo "  -f, --file FILE          Path to your Docker Compose file (default: docker-compose.yml)"
    echo "  -t, --token TOKEN        Your DigitalOcean API token (optional, only needed for DOCR)"
    echo "  -b, --backup-dir DIR     Directory for backups (default: backups)"
    echo "  -i, --interval TIME      Check interval (default: 1m)"
    echo "  -q, --quiet              Disable verbose logging"
    echo "  -c, --config             Create a sample dosync.yaml config file"
    echo "  -h, --help               Display this help and exit"
    exit 1
}

# Parse command-line arguments
CREATE_CONFIG=false
while [[ $# -gt 0 ]]; do
    key="$1"
    case $key in
    -f | --file)
        COMPOSE_FILE="$2"
        shift 2
        ;;
    -t | --token)
        DO_TOKEN="$2"
        shift 2
        ;;
    -b | --backup-dir)
        BACKUP_DIR="$2"
        shift 2
        ;;
    -i | --interval)
        CHECK_INTERVAL="$2"
        shift 2
        ;;
    -q | --quiet)
        VERBOSE=""
        shift
        ;;
    -c | --config)
        CREATE_CONFIG=true
        shift
        ;;
    -h | --help)
        show_help
        ;;
    *)
        echo -e "${RED}Error: Unknown option $1${NC}"
        show_help
        ;;
    esac
done

echo -e "${GREEN}Welcome to DOSync installer!${NC}"
echo -e "This will add DOSync to your Docker Compose project."
echo

# Check if the Docker Compose file exists
if [ ! -f "$COMPOSE_FILE" ]; then
    echo -e "${RED}Error: Docker Compose file '$COMPOSE_FILE' not found${NC}"
    echo -e "${YELLOW}Make sure you're running this script in your project directory with docker-compose.yml${NC}"
    echo -e "${YELLOW}Or specify a different file with -f/--file option${NC}"
    exit 1
fi

# Create backup directory if it doesn't exist
if [ ! -d "$BACKUP_DIR" ]; then
    echo -e "${YELLOW}Creating backup directory: $BACKUP_DIR${NC}"
    mkdir -p "$BACKUP_DIR"
fi

# Create backup of original Docker Compose file
BACKUP_FILE="${BACKUP_DIR}/$(basename ${COMPOSE_FILE}).$(date +%Y%m%d%H%M%S).bak"
echo -e "${YELLOW}Creating backup of ${COMPOSE_FILE} to ${BACKUP_FILE}${NC}"
cp "$COMPOSE_FILE" "$BACKUP_FILE"

# Create a temporary .env file if token is provided
if [ -n "$DO_TOKEN" ]; then
    echo -e "${YELLOW}Creating .env file with DO_TOKEN${NC}"
    echo "DO_TOKEN=$DO_TOKEN" >.env
    echo "CHECK_INTERVAL=$CHECK_INTERVAL" >>.env
    if [ -n "$VERBOSE" ]; then
        echo "VERBOSE=$VERBOSE" >>.env
    fi
else
    # Check if we need a DO_TOKEN (only if there are DOCR images)
    if grep -q "registry.digitalocean.com" "$COMPOSE_FILE"; then
        echo -e "${YELLOW}Your compose file contains DigitalOcean Container Registry images.${NC}"
        echo -e "${YELLOW}You'll need to create a .env file with your DigitalOcean API token.${NC}"
    else
        echo -e "${YELLOW}No DigitalOcean Container Registry images detected.${NC}"
        echo -e "${YELLOW}You can configure registry credentials in dosync.yaml${NC}"
    fi
fi

# Create config file if requested
if [ "$CREATE_CONFIG" = true ]; then
    echo -e "${YELLOW}Creating sample dosync.yaml configuration file...${NC}"
    cat >dosync.yaml <<EOF
# DOSync Configuration
# See https://github.com/localrivet/dosync for detailed documentation

# Global settings
checkInterval: "${CHECK_INTERVAL}"
verbose: true

# Registry configurations
registry:
  # Docker Hub configuration
  dockerhub:
    # For private repositories (optional)
    #username: ""
    #password: ""
    imagePolicy:
      # Example policy for semver tags
      filterTags:
        pattern: "v(\\d+\\.\\d+\\.\\d+)"
        extract: "$1"
      policy:
        semver: {}

  # GitHub Container Registry
  #ghcr:
  #  token: "" # GitHub Personal Access Token with read:packages scope
  #  imagePolicy:
  #    filterTags:
  #      pattern: "v(\\d+\\.\\d+\\.\\d+)"
  #    policy:
  #      semver: {}

  # DigitalOcean Container Registry
  #docr:
  #  token: "${DO_TOKEN}" # DigitalOcean API token
  #  imagePolicy:
  #    filterTags:
  #      pattern: "main-(\\d+)"
  #    policy:
  #      numerical:
  #        order: "desc"

  # Google Container Registry
  #gcr:
  #  credentialsFile: "/path/to/credentials.json"

  # Azure Container Registry
  #acr:
  #  registry: "myregistry.azurecr.io"
  #  clientID: ""
  #  clientSecret: ""

  # Amazon Elastic Container Registry
  #ecr:
  #  registry: "123456789012.dkr.ecr.region.amazonaws.com"
  #  awsAccessKeyID: ""
  #  awsSecretAccessKey: ""
  #  region: "us-east-1"

  # Harbor Registry
  #harbor:
  #  url: "https://harbor.example.com"
  #  username: ""
  #  password: ""

  # Quay.io Registry
  #quay:
  #  token: ""

  # Custom Registry
  #custom:
  #  url: "https://registry.example.com"
  #  username: ""
  #  password: ""
EOF
    echo -e "${GREEN}Created sample configuration in dosync.yaml${NC}"
fi

# Check if the Docker Compose file already has a dosync service
if grep -q "dosync:" "$COMPOSE_FILE"; then
    echo -e "${YELLOW}Warning: dosync service already exists in $COMPOSE_FILE${NC}"
    echo -e "${YELLOW}Skipping service addition${NC}"
else
    # Determine the indent level for services
    INDENT=$(grep -E '^ +[a-zA-Z0-9_-]+:' "$COMPOSE_FILE" | head -1 | sed -E 's/^( +).*/\1/')

    # If no indent level found, use default of 2 spaces
    if [ -z "$INDENT" ]; then
        INDENT="  "
    fi

    echo -e "${GREEN}Adding dosync service to $COMPOSE_FILE${NC}"

    # Add the dosync service to the Docker Compose file
    cat >>"$COMPOSE_FILE" <<EOF

${INDENT}# Self-updating DOSync service
${INDENT}dosync:
${INDENT}  image: localrivet/dosync:latest
${INDENT}  restart: unless-stopped
${INDENT}  volumes:
${INDENT}    - /var/run/docker.sock:/var/run/docker.sock
${INDENT}    - ./${COMPOSE_FILE}:/app/docker-compose.yml
${INDENT}    - ./${BACKUP_DIR}:/app/backups
EOF

    # Add config file volume if created
    if [ "$CREATE_CONFIG" = true ]; then
        echo "${INDENT}    - ./dosync.yaml:/app/dosync.yaml" >>"$COMPOSE_FILE"
    fi

    # Add environment variables
    echo "${INDENT}  environment:" >>"$COMPOSE_FILE"

    # Only add DO_TOKEN if we have DOCR images
    if grep -q "registry.digitalocean.com" "$COMPOSE_FILE"; then
        echo "${INDENT}    - DO_TOKEN=\${DO_TOKEN}" >>"$COMPOSE_FILE"
    fi

    echo "${INDENT}    - CHECK_INTERVAL=\${CHECK_INTERVAL:-${CHECK_INTERVAL}}" >>"$COMPOSE_FILE"

    # Add VERBOSE environment variable if enabled
    if [ -n "$VERBOSE" ]; then
        echo "${INDENT}    - VERBOSE=\${VERBOSE:-$VERBOSE}" >>"$COMPOSE_FILE"
    fi

    # Add networks section - try to detect existing networks
    NETWORK=$(grep -E 'networks:' -A 10 "$COMPOSE_FILE" | grep -E '^ +[a-zA-Z0-9_-]+:' | head -1 | sed -E 's/^ +([a-zA-Z0-9_-]+):.*/\1/')

    if [ -n "$NETWORK" ]; then
        echo -e "${GREEN}Using existing network: $NETWORK${NC}"
        echo "${INDENT}  networks:" >>"$COMPOSE_FILE"
        echo "${INDENT}    - $NETWORK" >>"$COMPOSE_FILE"
    fi
fi

echo
echo -e "${GREEN}DOSync installation complete!${NC}"
echo -e "${YELLOW}Supported container registries:${NC}"
echo "  • Docker Hub"
echo "  • GitHub Container Registry (GHCR)"
echo "  • Google Container Registry (GCR)"
echo "  • Azure Container Registry (ACR)"
echo "  • DigitalOcean Container Registry (DOCR)"
echo "  • Amazon Elastic Container Registry (ECR)"
echo "  • Harbor"
echo "  • Quay.io"
echo "  • Custom Docker-compatible registries"
echo
echo -e "${YELLOW}Next steps:${NC}"

if grep -q "registry.digitalocean.com" "$COMPOSE_FILE"; then
    echo -e "1. Make sure you have a ${GREEN}.env${NC} file with your DigitalOcean API token:"
    echo -e "   ${GREEN}DO_TOKEN=${NC}your_digitalocean_token_here"
    echo
fi

if [ "$CREATE_CONFIG" = true ]; then
    echo -e "1. Edit ${GREEN}dosync.yaml${NC} to configure your registry authentication and image policies"
    echo
fi

echo -e "2. Start DOSync with:"
echo -e "   ${GREEN}docker compose -f $COMPOSE_FILE up -d dosync${NC}"
echo
echo -e "3. Check logs with:"
echo -e "   ${GREEN}docker compose -f $COMPOSE_FILE logs -f dosync${NC}"
echo
echo -e "${YELLOW}For help and documentation, visit:${NC} ${GREEN}https://github.com/localrivet/dosync${NC}"
