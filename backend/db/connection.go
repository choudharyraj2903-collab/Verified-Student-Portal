package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
	"student_portal/backend/config"
)

type DB struct {
	pool *pgxpool.Pool
	cfg  *config.DatabaseConfig
}

type Tx = pgx.Tx

// func NewDB(cfg *config.databaseConfig) (*DB, error) {

// 	pgxconfig ,err := pgx.ParseConfig(cfg.DSN())
// 	if err!=nil{
// 		return nil, fmt.Errorf("failed to parse database config: %v", err)
// 	}

// 	poolConfig.MaxConns        = int32(cfg.MaxOpenConns)
// 	poolConfig.MinConns        = int32(cfg.MaxIdleConns)
//     poolConfig.MaxConnLifetime = time.Duration(cfg.ConnMaxLifetimeMinutes) * time.Minute
//     poolConfig.MaxConnIdleTime = 10 * time.Minute
//     poolConfig.HealthCheckPeriod = 1 * time.Minute

// 	poolConfig.ConnConfig.ConnectTimeout = 10 * time.Second

// 	poolconfig.BeforeAcquire = func(ctx context.Context, conn *pgx.Conn) bool {
// 		if err := conn.Ping(ctx); err != nil {
// 			fmt.Printf("Connection failed health check: %v\n", err)
// 			return false // Don't use this connection
// 		}
// 		return true // Use this connection
// 	}

// 	poolconfig.AfterRelease = func(conn *pgx.Conn) bool {
// 		// called every time a connection is returned to the pool
//     // return false to discard it instead of returning it
//     // discard connections that have been alive too long
// 		if time.Since(conn.PgConn().LastUsed()) > 30*time.Minute {
// 			fmt.Printf("Connection discarded after release due to age: %v\n", conn.PgConn().LastUsed())
// 			return false // Discard this connection
// 		}
// 		return true // Return this connection
// 	}

// 	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create database connection pool: %v", err)
// 	}

// 	// Validate the connection immediately after creating the pool
// 	if err := validate(pool); err != nil {
// 		pool.Close()
// 		return nil, fmt.Errorf("database connection validation failed: %v", err)
// 	}
// 	log.Println("Database connection pool created and validated successfully")
// 	log.Printf("DB pool created — maxOpen: %d, minIdle: %d, lifetime: %dm",
//     cfg.MaxOpenConns, cfg.MaxIdleConns, cfg.ConnMaxLifetimeMinutes)

// 	return &DB{pool: pool, cfg: cfg}, nil

// }

func NewDB(cfg *config.DatabaseConfig) (*DB, error) {
	constructingDSN := fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.DB_USER, cfg.DB_PASSWORD, cfg.DB_HOST, cfg.DB_PORT, cfg.DB_NAME, cfg.DB_SSL_MODE)
	poolConfig, err := pgxpool.ParseConfig(constructingDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %v", err)
	}

	poolConfig.MaxConns = int32(cfg.DB_MAX_OPEN_CONNS)
	poolConfig.MinConns = int32(cfg.DB_MAX_IDLE_CONNS)
	poolConfig.MaxConnLifetime = time.Duration(cfg.DB_CONN_MAX_LIFETIME_MINUTES) * time.Minute
	poolConfig.MaxConnIdleTime = 10 * time.Minute
	poolConfig.HealthCheckPeriod = 1 * time.Minute
	poolConfig.ConnConfig.ConnectTimeout = 10 * time.Second

	poolConfig.BeforeAcquire = func(ctx context.Context, conn *pgx.Conn) bool {
		return conn.Ping(ctx) == nil
	}
	poolConfig.AfterRelease = func(conn *pgx.Conn) bool {
		return true // Let the pool decide whether to keep or discard based on MaxConnLifetime and MaxConnIdleTime
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create database connection pool: %v", err)
	}

	log.Printf("DB pool created — maxOpen: %d, minIdle: %d, lifetime: %dm",
		cfg.DB_MAX_OPEN_CONNS, cfg.DB_MAX_IDLE_CONNS, cfg.DB_CONN_MAX_LIFETIME_MINUTES)

	return &DB{pool: pool, cfg: cfg}, nil
}

func validate(pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Always use the timeout for pinging the database to avoid hanging
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("database unreachable at localhost:5432 — connection refused: %v", err)
	}

	stats := pool.Stat()
	log.Printf("Database connection pool stats: TotalConns=%d, IdleConns=%d, AcquiredConns=%d, MaxConns=%d",
		stats.TotalConns(), stats.IdleConns(), stats.AcquiredConns(), stats.MaxConns())
	return nil
}

type HealthStatus struct {
	Database      string // "ok" or "unreachable"
	LatencyMs     int64  // ping latency in ms
	TotalConns    int32  // total connections in pool
	IdleConns     int32  // idle connections
	AcquiredConns int32  // connections currently in use
	Error         string // empty if ok, error message if not
}

// func HealthCheck(ctx context.Context) (*HealthStatus, error) {

// 	start:=time.Now()
// // Call validate(db.pool)
// // Pass the incoming ctx — the caller controls the timeout. The health endpoint should set a 3 second timeout on its context before calling this

//     err := validate(db.pool)
// 	latency := time.Since(start).Milliseconds()

// 	status := db.pool.Stat()

// 	status := &HealthStatus{}
// 		LatencyMs:     latency,
// 		TotalConns:    status.TotalConns(),
// 		IdleConns:     status.IdleConns(),
// 		AcquiredConns: status.AcquiredConns(),
// 		Error:         "",

// 		if err != nil {
// 			status.Database = "unreachable"
// 			status.Error = err.Error()
// 		}
// 		else {
// 			status.Database = "ok"
// 			status.Error = ""
// 		}
// 	return status, nil
// }

func (db *DB) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	start := time.Now()
	err := validate(db.pool)
	latency := time.Since(start).Milliseconds()

	stats := db.pool.Stat()
	status := &HealthStatus{
		LatencyMs:     latency,
		TotalConns:    stats.TotalConns(),
		IdleConns:     stats.IdleConns(),
		AcquiredConns: stats.AcquiredConns(),
	}

	if err != nil {
		status.Database = "unreachable"
		status.Error = err.Error()
	} else {
		status.Database = "ok"
	}
	return status, nil
}

// func (db *DB) WithTransaction(ctx context.Context, fn func(tx pgx.Tx) error) error {

// 	tx, err := db.pool.Begin(ctx)
// 	if err != nil {
// 		return fmt.Errorf("failed to begin transaction: %v", err)
// 	}
// 	defer func() {
// 		if p := recover(); p != nil {
// 			tx.Rollback(ctx)
// 			panic(p) // re-throw panic after Rollback
// 		} else if err != nil {
// 			tx.Rollback(ctx) // err is non-nil; don't change it
// 		} else {
// 			err = tx.Commit(ctx) // err is nil; if Commit returns error update err
// 		}
// 	}()

//		if err := tx.Commit(ctx); err != nil {
//	    return fmt.Errorf("failed to commit transaction: %w", err)
//	}
//
// }
func (db *DB) WithTransaction(ctx context.Context, fn func(tx pgx.Tx) error) (err error) {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback(ctx)
			panic(p)
		} else if err != nil {
			tx.Rollback(ctx)
		} else {
			err = tx.Commit(ctx)
		}
	}()

	err = fn(tx)
	return err
}

func (db *DB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return db.pool.Query(ctx, sql, args...)
}

func (db *DB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return db.pool.QueryRow(ctx, sql, args...)
}

func (db *DB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return db.pool.Exec(ctx, sql, args...)
}

func (db *DB) Pool() *pgxpool.Pool {
	return db.pool
}

func (db *DB) Close() {
	log.Println("closing database connection pool...")
	db.pool.Close()

	status := db.pool.Stat()
	log.Printf("Database connection pool stats at close: TotalConns=%d, IdleConns=%d, AcquiredConns=%d, MaxConns=%d",
		status.TotalConns(), status.IdleConns(), status.AcquiredConns(), status.MaxConns())
}
