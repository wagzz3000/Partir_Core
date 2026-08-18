#!/bin/bash
set -e

# 1. Install Docker
if ! command -v docker &> /dev/null; then
    echo "Installing Docker..."
    curl -fsSL https://get.docker.com -o get-docker.sh
    sh get-docker.sh
    rm get-docker.sh
else
    echo "Docker already installed."
fi

# 2. Setup Directories
mkdir -p /root/partir-core
mkdir -p /root/Partir_Plugins

# 3. Unpack Core
if [ -f /root/core.tar.gz ]; then
    echo "Unpacking Core..."
    tar -xzf /root/core.tar.gz -C /root/partir-core
fi

# 4. Unpack Plugins
if [ -f /root/plugins.tar.gz ]; then
    echo "Unpacking Plugins..."
    tar -xzf /root/plugins.tar.gz -C /root/Partir_Plugins
fi

# 5. Unpack Secrets
if [ -f /root/secrets.tar.gz ]; then
    echo "Unpacking Secrets..."
    mkdir -p /root/partir-core/secrets
    tar -xzf /root/secrets.tar.gz -C /root/partir-core/secrets
fi

# 6. Create production environment file if missing
if [ ! -f /root/partir-core/.env ]; then
    echo "Creating .env from example..."
    cp /root/partir-core/.env.example /root/partir-core/.env
fi
