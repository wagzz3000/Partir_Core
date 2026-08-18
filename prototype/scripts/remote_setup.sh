#!/bin/bash
set -e

echo "═══════════════════════════════════════════════════"
echo "  Partir Core — Server Setup (Ubuntu)"
echo "═══════════════════════════════════════════════════"

# Update and install basic deps
apt-get update
apt-get install -y ca-certificates curl gnupg lsb-release openssl

# Install Docker (if not present)
if ! command -v docker &> /dev/null; then
    echo "Installing Docker..."
    mkdir -p /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg \
        | gpg --dearmor -o /etc/apt/keyrings/docker.gpg --yes

    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" \
        | tee /etc/apt/sources.list.d/docker.list > /dev/null

    apt-get update
    apt-get install -y docker-ce docker-ce-cli containerd.io \
        docker-buildx-plugin docker-compose-plugin
else
    echo "Docker already installed, skipping"
fi

# Install Go (if not present)
GO_VERSION="1.23.0"
if ! command -v /usr/local/go/bin/go &> /dev/null; then
    echo "Installing Go ${GO_VERSION}..."
    curl -LO "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz"
    rm -rf /usr/local/go && tar -C /usr/local -xzf "go${GO_VERSION}.linux-amd64.tar.gz"
    rm -f "go${GO_VERSION}.linux-amd64.tar.gz"

    # Add to PATH permanently
    grep -q '/usr/local/go/bin' /root/.bashrc || \
        echo 'export PATH=$PATH:/usr/local/go/bin' >> /root/.bashrc
    export PATH=$PATH:/usr/local/go/bin
else
    echo "Go already installed, skipping"
fi

# Verification
echo ""
echo "Installed versions:"
docker version --format '  Docker: {{.Server.Version}}'
docker compose version
echo "  $(go version)"
echo ""
echo "═══════════════════════════════════════════════════"
echo "  ✓ Server setup complete"
echo "  Next: upload code and run remote_boot.sh"
echo "═══════════════════════════════════════════════════"
