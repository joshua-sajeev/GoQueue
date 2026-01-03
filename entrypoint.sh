#!/bin/bash

# Construct database URL from env variables
DATABASE_URL="postgres://$POSTGRES_USER:$POSTGRES_PASSWORD@$POSTGRES_HOST:$POSTGRES_PORT/$POSTGRES_DB?sslmode=disable"

# Run migrations (only once, from API service)
if [ "$SERVICE_TYPE" = "api" ]; then
  echo "Running database migrations..."
  goose -dir ./migrations postgres "$DATABASE_URL" up
fi

# Start appropriate Air config based on service type
if [ "$SERVICE_TYPE" = "worker" ]; then
  echo "Starting worker service with hot-reload..."
  air -c .air-worker.toml
else
  echo "Starting API service with hot-reload..."
  air -c .air-api.toml
fi
