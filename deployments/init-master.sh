#!/bin/bash
set -e

SUBNET=$(hostname -i | awk -F. '{print $1"."$2".0.0/16"}')

if [ -z "$PGPASSWORD" ]; then
    echo "Ошибка: Переменная PGPASSWORD не задана!"
    exit 1
fi


# 1. Create the replication user inside Postgres
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE USER $PGUSER WITH REPLICATION ENCRYPTED PASSWORD '${PGPASSWORD}';
EOSQL

# 2. Append replication permissions to pg_hba.conf dynamically
echo "host replication $PGUSER $SUBNET md5" >> "$PGDATA/pg_hba.conf"