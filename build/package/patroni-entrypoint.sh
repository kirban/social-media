#!/bin/bash
set -e

# Map DB_USER / DB_PASSWORD from .env to patroni's superuser credentials.
# Patroni reads these from the YAML we generate below, but we also expose them
# for the post_init script that creates the application database.
export PATRONI_SUPERUSER_USERNAME="${DB_USER}"
export PATRONI_SUPERUSER_PASSWORD="${DB_PASSWORD}"
export PATRONI_REPLICATION_USERNAME="replicator"
export PATRONI_REPLICATION_PASSWORD="${REPLICATION_PASSWORD:-replicator_pass}"

PGDATA="${PGDATA:-/var/lib/postgresql/data}"

mkdir -p "${PGDATA}" /scripts
chown -R postgres:postgres /var/lib/postgresql
chmod 700 "${PGDATA}"

cat > /etc/patroni.yml <<EOF
scope: ${PATRONI_SCOPE:-sm-cluster}
namespace: /service/
name: ${PATRONI_NAME}

restapi:
  listen: 0.0.0.0:8008
  connect_address: ${PATRONI_NAME}:8008

etcd3:
  hosts: ${PATRONI_ETCD_HOSTS:-sm-etcd:2379}

bootstrap:
  dcs:
    ttl: 30
    loop_wait: 10
    retry_timeout: 10
    maximum_lag_on_failover: 1048576
    synchronous_mode: quorum
    synchronous_node_count: 1
    postgresql:
      use_pg_rewind: true
      use_slots: true
      parameters:
        wal_level: replica
        hot_standby: "on"
        max_wal_senders: 10
        max_replication_slots: 10
        wal_keep_size: 1024MB
        synchronous_commit: "on"
  initdb:
    - encoding: UTF8
    - data-checksums
  pg_hba:
    - host replication replicator 0.0.0.0/0 md5
    - host all all 0.0.0.0/0 md5
  post_init: bash /scripts/post-init.sh

postgresql:
  listen: 0.0.0.0:5432
  connect_address: ${PATRONI_NAME}:5432
  data_dir: ${PGDATA}
  bin_dir: /usr/lib/postgresql/18/bin
  pgpass: /tmp/pgpass0
  pg_hba:
    - host replication replicator 0.0.0.0/0 md5
    - host all all 0.0.0.0/0 scram-sha-256
  authentication:
    replication:
      username: replicator
      password: ${PATRONI_REPLICATION_PASSWORD}
    superuser:
      username: ${PATRONI_SUPERUSER_USERNAME}
      password: ${PATRONI_SUPERUSER_PASSWORD}
    rewind:
      username: ${PATRONI_SUPERUSER_USERNAME}
      password: ${PATRONI_SUPERUSER_PASSWORD}

tags:
  nofailover: false
  noloadbalance: false
  clonefrom: false
  nosync: false
EOF

exec gosu postgres patroni /etc/patroni.yml
