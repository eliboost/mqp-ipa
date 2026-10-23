#!/bin/sh
set -e  # Exit on error

echo "========================================="
echo "ESProxy Setup Script"
echo "========================================="

# Get the directory where the script is located
# shellcheck disable=SC3028
# shellcheck disable=SC3054
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$SCRIPT_DIR"

echo ""
echo "1 | Building frontend..."
cd "$PROJECT_ROOT/frontend"
npm run build-local-with-auth

# Verify dist folder was created
if [ ! -d "$PROJECT_ROOT/frontend/dist" ]; then
    echo "❌ Error: Frontend build failed - dist folder not found!"
    exit 1
fi

echo "✅ Frontend build successful"

echo ""
echo "2 | Building backend..."
cd "$PROJECT_ROOT/backend"
mvn -DskipTests clean package

echo "✅ Backend build successful"

echo ""
echo "3 | Stopping existing Docker containers..."
cd "$PROJECT_ROOT/esproxy-image"
docker compose down || true

echo ""
echo "4 | Starting Docker containers..."
docker compose up --build -d

sleep 5

echo ""
echo "========================================="
echo "✅ ESProxy is starting up!"
echo "========================================="
echo ""
echo "Login URL:   http://localhost:8289/config/login.jsp"
echo "App URL:     http://localhost:8289/mqp-ops-ui/"
echo ""
echo "========================================="