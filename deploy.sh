#!/bin/bash

# Navigate to the app directory
cd /opt/app

# Pull the latest code
echo "--- Pulling latest changes from Git ---"
git pull

# Build the new image
echo "--- Building Docker image ---"
docker build -t symptomstracker .

# Stop and remove the old container if it exists
echo "--- Cleaning up old container ---"
docker stop symptomstracker 2>/dev/null || true
docker rm symptomstracker 2>/dev/null || true

# Run the new container
echo "--- Starting new container ---"
docker run -d \
  --name symptomstracker \
  -p 8080:8080 \
  -v /opt/app/data:/app/data \
  -v /opt/whisper/whisper.cpp:/whisper \
  -e AUTH_SESSION_SECRET="OpxOYqd7dUZ4IJmXNJ1n62sv57cJG8tS/QJLiZd0SK8=" \
  -e WEBAUTHN_RP_ORIGINS=https://diary.airberry.no \
  -e WEBAUTHN_RP_ID=diary.airberry.no \
  symptomstracker

echo "--- Deployment complete! ---"
docker ps | grep symptomstracker