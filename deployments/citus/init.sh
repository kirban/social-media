#!/bin/bash
# One-shot init for the Citus cluster. It:
#   1. waits for the coordinator and the two workers to accept connections,
#   2. registers the workers on the coordinator (idempotent),
#   3. waits until the app has created its schema (dialog_message exists),
#   4. applies deployments/citus/distribute.sql to shard the tables.
#
# Runs inside a small postgres image (needs `psql`). Auth is `trust` on the
# private compose network (dev only), so no password is required.
set -euo pipefail

COORD="${COORDINATOR_HOST:-citus-coordinator}"
WORKERS=(citus-worker-1 citus-worker-2)

coord() { psql -h "$COORD" -p 5432 -U "${DB_USER}" -d "${DB_NAME}" -tAc "$1"; }

echo "=== [citus-init] waiting for coordinator ==="
until coord "SELECT 1" >/dev/null 2>&1; do sleep 2; done

for w in "${WORKERS[@]}"; do
  echo "=== [citus-init] waiting for worker ${w} ==="
  until psql -h "$w" -p 5432 -U "${DB_USER}" -d "${DB_NAME}" -tAc "SELECT 1" >/dev/null 2>&1; do sleep 2; done

  if [ "$(coord "SELECT count(*) FROM pg_dist_node WHERE nodename='${w}'")" -eq 0 ]; then
    echo "=== [citus-init] registering worker ${w} ==="
    coord "SELECT citus_add_node('${w}', 5432)" >/dev/null
  else
    echo "=== [citus-init] worker ${w} already registered ==="
  fi
done

echo "=== [citus-init] waiting for the app to create the dialog_message table ==="
until [ "$(coord "SELECT to_regclass('public.dialog_message') IS NOT NULL")" = "t" ]; do sleep 2; done

# Skip if dialog_message is already distributed (idempotent re-runs).
if [ "$(coord "SELECT count(*) FROM pg_dist_partition WHERE logicalrelid = 'dialog_message'::regclass")" -ge 1 ]; then
  echo "=== [citus-init] dialog_message already distributed — nothing to do ==="
  exit 0
fi

echo "=== [citus-init] applying distribute.sql ==="
psql -v ON_ERROR_STOP=1 -h "$COORD" -p 5432 -U "${DB_USER}" -d "${DB_NAME}" -f /distribute.sql

echo "=== [citus-init] done. Shard layout: ==="
coord "SELECT nodename, count(*) FROM citus_shards WHERE table_name='dialog_message'::regclass GROUP BY nodename"
