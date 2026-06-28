package db

import (
	"database/sql"
	"errors"
	"fmt"
	"sync/atomic"

	_ "github.com/lib/pq"

	"github.com/kirban/social-media/internal/config"
)

// Cluster holds one master connection for writes and zero-or-more replica
// connections for reads. When no replicas are configured, reads fall back to
// the master.
type Cluster struct {
	master   *DB
	replicas []*sql.DB
	rrIdx    uint64
}

func NewCluster(cfg config.DBConfig) (*Cluster, error) {
	master, err := New(cfg)
	if err != nil {
		return nil, fmt.Errorf("master: %w", err)
	}

	replicas := make([]*sql.DB, 0, len(cfg.Replicas))
	for i, rc := range cfg.Replicas {
		r, err := openReplica(cfg, rc)
		if err != nil {
			return nil, fmt.Errorf("replica[%d]: %w", i, err)
		}
		replicas = append(replicas, r)
	}

	return &Cluster{master: master, replicas: replicas}, nil
}

func openReplica(base config.DBConfig, rc config.ReplicaConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
		rc.Host, rc.Port, base.DBName, base.Username, base.Password, base.SSLMode,
	)

	r, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}

	if err := r.Ping(); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}

	r.SetMaxOpenConns(base.MaxOpenConns)
	r.SetMaxIdleConns(base.MaxIdleConns)
	r.SetConnMaxLifetime(base.MaxConnLifetime)

	return r, nil
}

// Master returns the master DB connection for write operations.
func (c *Cluster) Master() *sql.DB { return c.master.DB }

// Replica returns a replica connection using round-robin selection.
// Falls back to master when no replicas are configured.
func (c *Cluster) Replica() *sql.DB {
	if len(c.replicas) == 0 {
		return c.master.DB
	}
	idx := atomic.AddUint64(&c.rrIdx, 1) % uint64(len(c.replicas))
	return c.replicas[idx]
}

func (c *Cluster) Migrate() error { return c.master.Migrate() }

func (c *Cluster) Close() error {
	var errs []error
	if err := c.master.Close(); err != nil {
		errs = append(errs, err)
	}
	for _, r := range c.replicas {
		if err := r.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
