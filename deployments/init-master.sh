#!/bin/bash
set -e

SUBNET=$(hostname -I | awk '{print $1}' | awk -F. '{print $1"."$2".0.0/16"}')

# 1. Create the replication user and configure synchronous replication
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE USER $REPL_USER WITH REPLICATION ENCRYPTED PASSWORD '${REPL_PASS}';
    ALTER SYSTEM SET synchronous_standby_names = 'FIRST 1 ("sm-db-sl1", "sm-db-sl2")';
    ALTER SYSTEM SET synchronous_commit = 'on';
EOSQL

# 2. Append replication permissions to pg_hba.conf dynamically
echo "host replication $REPL_USER $SUBNET md5" >> "$PGDATA/pg_hba.conf"