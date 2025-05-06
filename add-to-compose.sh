#!/bin/bash

set -e

# Colors for visual feedback
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}DOSync Setup Tool${NC}"
echo -e "This script will add DOSync to your Docker Compose file."
echo

# Ensure the user has a docker-compose.yml file
if [ ! -f docker-compose.yml ]; then
    echo -e "${RED}Error: docker-compose.yml not found in current directory.${NC}"
    echo "Please run this script from the directory containing your docker-compose.yml file."
    exit 1
fi

# Ask for the compose file name (default: docker-compose.yml)
read -p "Docker Compose filename [docker-compose.yml]: " COMPOSE_FILE
COMPOSE_FILE=${COMPOSE_FILE:-docker-compose.yml}

if [ ! -f "$COMPOSE_FILE" ]; then
    echo -e "${RED}Error: $COMPOSE_FILE does not exist.${NC}"
    exit 1
fi

# Ask for the check interval (default: 5 minutes)
read -p "How often should DOSync check for updates? [5m]: " CHECK_INTERVAL
CHECK_INTERVAL=${CHECK_INTERVAL:-5m}

# Determine if the user should create a config file
echo
echo -e "${YELLOW}Registry Configuration:${NC}"
echo "DOSync supports multiple container registries:"
echo "1. Docker Hub"
echo "2. GitHub Container Registry (GHCR)"
echo "3. Google Container Registry (GCR)"
echo "4. Azure Container Registry (ACR)"
echo "5. DigitalOcean Container Registry (DOCR)"
echo "6. Amazon Elastic Container Registry (ECR)"
echo "7. Harbor"
echo "8. Quay.io"
echo "9. Custom Docker-compatible registries"

read -p "Do you want to create a dosync.yaml config file for registry authentication? [y/N]: " CREATE_CONFIG
CREATE_CONFIG=${CREATE_CONFIG:-n}

if [[ $CREATE_CONFIG =~ ^[Yy]$ ]]; then
    echo "Creating dosync.yaml config file..."

    # Create a basic config template
    cat >dosync.yaml <<EOF
# DOSync Configuration
registry:
  # Docker Hub configuration (public images only by default)
  dockerhub:
    # Optional authentication for private repos
    # username: "your-username"
    # password: "your-password"
    # Image policy configuration
    # imagePolicy:
    #   filterTags:
    #     pattern: "v(\\d+\\.\\d+\\.\\d+)"
    #     extract: "$1"
    #   policy:
    #     semver:
    #       range: ">=1.0.0"

  # GitHub Container Registry configuration
  # ghcr:
  #   token: "your-github-token"
  #   imagePolicy:
  #     filterTags:
  #       pattern: "v(\\d+\\.\\d+\\.\\d+)"
  #     policy:
  #       semver: {}

  # DigitalOcean Container Registry configuration
  # docr:
  #   token: ${DO_TOKEN:-"your-do-token"}
  #   imagePolicy:
  #     filterTags:
  #       pattern: "main-(\\d+)"
  #     policy:
  #       numerical:
  #         order: "desc"

  # Google Container Registry configuration
  # gcr:
  #   credentialsFile: "/path/to/credentials.json"
  #   imagePolicy:
  #     policy:
  #       alphabetical:
  #         order: "desc"

  # Add configurations for other registries as needed
EOF

    echo -e "${GREEN}Created dosync.yaml with example configuration.${NC}"
    echo "Please edit this file to add your specific registry credentials and image policies."
fi

# Create the DOSync service configuration
DOSYNC_SERVICE=$(
    cat <<-EOF

  dosync:
    image: rnd/dosync:latest
    restart: unless-stopped
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./docker-compose.yml:/app/docker-compose.yml
EOF
)

# Conditionally add config volume if created
if [[ $CREATE_CONFIG =~ ^[Yy]$ ]]; then
    DOSYNC_SERVICE+=$(
        cat <<-EOF
      - ./dosync.yaml:/app/dosync.yaml
EOF
    )
fi

# Add environment variables
DOSYNC_SERVICE+=$(
    cat <<-EOF
    environment:
      - CHECK_INTERVAL=$CHECK_INTERVAL
EOF
)

# If the user has DOCR images, suggest adding DO_TOKEN
DOCR_IMAGES=$(grep -c "registry.digitalocean.com" $COMPOSE_FILE || true)
if [ "$DOCR_IMAGES" -gt 0 ]; then
    echo -e "${YELLOW}Notice:${NC} Your compose file contains DigitalOcean Container Registry images."
    read -p "Add DO_TOKEN environment variable? [Y/n]: " ADD_DO_TOKEN
    ADD_DO_TOKEN=${ADD_DO_TOKEN:-y}

    if [[ $ADD_DO_TOKEN =~ ^[Yy]$ ]]; then
        read -p "Enter your DigitalOcean API token: " DO_TOKEN
        if [ -n "$DO_TOKEN" ]; then
            DOSYNC_SERVICE+=$(
                cat <<-EOF
      - DO_TOKEN=$DO_TOKEN
EOF
            )
        else
            echo -e "${YELLOW}Warning:${NC} No token provided. You will need to add DO_TOKEN later."
        fi
    fi
fi

# Add the DOSYNC service to the compose file
echo "$DOSYNC_SERVICE" >>$COMPOSE_FILE

echo -e "${GREEN}Successfully added DOSync to $COMPOSE_FILE${NC}"
echo
echo -e "${YELLOW}Next steps:${NC}"
echo "1. Review the configuration in $COMPOSE_FILE"
if [[ $CREATE_CONFIG =~ ^[Yy]$ ]]; then
    echo "2. Edit dosync.yaml to configure your registry authentication and image policies"
else
    echo "2. Consider creating a dosync.yaml file for registry authentication and image policies"
fi
echo "3. Start the service with: docker-compose up -d dosync"
echo
echo -e "${GREEN}DOSync will now automatically keep your services up to date with the latest container images.${NC}"
