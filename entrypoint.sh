#!/bin/bash

# Construct database URL from env variables
DATABASE_URL="postgres://$POSTGRES_USER:$POSTGRES_PASSWORD@$POSTGRES_HOST:$POSTGRES_PORT/$POSTGRES_DB?sslmode=disable"

# Run migrations
goose -dir ./migrations postgres "$DATABASE_URL" up

# Start Air
air
