#!/bin/bash
set -e

echo "=========================================="
echo "    STD GAME BOT — VPS INSTALLATION       "
echo "  Developed by STD DEEPANSHU (StdBots)    "
echo "=========================================="

if [ "$EUID" -ne 0 ]; then 
  echo "Please run as root or with sudo"
  exit 1
fi

apt-get update && apt-get install -y git curl wget ca-certificates tzdata

# Check docker
if ! command -v docker &> /dev/null; then
    echo "Installing Docker..."
    curl -fsSL https://get.docker.com -o get-docker.sh
    sh get-docker.sh
fi

if ! command -v docker-compose &> /dev/null; then
    echo "Installing Docker Compose..."
    apt-get install -y docker-compose-plugin
fi

echo "Deploying via Docker Compose..."
docker compose up -d --build

echo "StdGameBot is now online!"
