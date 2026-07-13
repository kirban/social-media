-- Citus sharding for the dialog domain.
--
-- Run ONCE against the Citus coordinator, AFTER:
--   1. the app has applied its goose migrations (tables exist), and
--   2. the worker nodes are registered (see pg_dist_node).
--
-- This file is intentionally kept OUT of the goose migration chain so that the
-- plain (non-Citus) local / replicated setups are not affected by Citus DDL.
--
-- Strategy: shard `dialog_message` by `dialog_id`; keep the small `users`,
-- `dialog` and `dialog_user` tables as REFERENCE tables (a full copy on every
-- worker) so that:
--   * ListMessages (WHERE dialog_id = $1)      -> hits a single shard
--   * GetDialogID  (user-pair lookup on dialog_user) -> resolved locally
--   * an entire conversation lives on ONE shard
--
-- Idempotency: create_reference_table / create_distributed_table error if the
-- table is already distributed, so the wrapping init script checks
-- pg_dist_partition before invoking this file.

CREATE EXTENSION IF NOT EXISTS citus;

-- Register the coordinator in the cluster metadata. This lets coordinator-local
-- tables hold foreign keys to reference tables (they become "Citus local"
-- tables automatically) and lets workers route back to the coordinator.
SELECT citus_set_coordinator_host('citus-coordinator', 5432);

-- Reference tables — created in FK-dependency order (referenced tables first).
-- Each becomes a single shard replicated to every worker node.
SELECT create_reference_table('users');
SELECT create_reference_table('dialog');
SELECT create_reference_table('dialog_user');

-- Citus requires the distribution column to be part of every PK / unique
-- constraint. Swap the single-column PK for a composite one that includes the
-- distribution column. This composite PK also serves as the replica identity
-- needed for NON-BLOCKING (logical-replication) shard moves during resharding.
ALTER TABLE "dialog_message" DROP CONSTRAINT dialog_message_pkey;
ALTER TABLE "dialog_message" ADD PRIMARY KEY (dialog_id, id);

-- Enough shards up front so we can rebalance across more workers later without
-- re-splitting. 32 shards / 2 workers = 16 each; adding a 3rd worker rebalances
-- to ~11 each.
SET citus.shard_count = 32;

-- Distribute the big table. Existing rows (if any) are moved into shards.
-- FKs dialog_message.dialog_id -> dialog and from/to -> users are
-- distributed->reference, which Citus allows.
SELECT create_distributed_table('dialog_message', 'dialog_id');
