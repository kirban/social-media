#!/bin/bash
set -e
# Patroni calls this script once after the initial cluster bootstrap (only on the leader).
# It receives connection args: --host H --port P --username U --dbname D
# We use them to authenticate and create the application database.
PGPASSWORD="${DB_PASSWORD}" psql "$@" \
    -c "CREATE DATABASE \"${DB_NAME}\" ENCODING 'UTF8';" \
    && echo "=== Database ${DB_NAME} created ===" \
    || echo "=== Database ${DB_NAME} already exists, skipping ==="
