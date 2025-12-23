package config

import "fmt"

// Database holds the database configuration
type Database struct {
	Username     string `envconfig:"DB_USERNAME"`
	Password     string `envconfig:"DB_PASSWORD"`
	Host         string `envconfig:"DB_HOST"`
	Port         string `envconfig:"DB_PORT"`
	Database     string `envconfig:"DB_DATABASE"`
	SSLMode      string `envconfig:"DB_SSL_MODE" default:"require"`
	PoolMaxConns int    `envconfig:"DB_POOL_MAX_CONNS" default:"10"`
}

// ToDbConnectionUri returns a connection URI to be used with the pgx package
func (d Database) ToDbConnectionUri() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s&pool_max_conns=%d",
		d.Username,
		d.Password,
		d.Host,
		d.Port,
		d.Database,
		d.SSLMode,
		d.PoolMaxConns,
	)
}

// ToMigrationUri returns a connection URI for golang-migrate with pgx5 driver
func (d Database) ToMigrationUri() string {
	return fmt.Sprintf("pgx5://%s:%s@%s:%s/%s?sslmode=%s",
		d.Username,
		d.Password,
		d.Host,
		d.Port,
		d.Database,
		d.SSLMode,
	)
}

// Server holds the configuration for the API server
type Server struct {
	ServerPort string `envconfig:"SERVER_PORT" default:"8080"`
	Database   Database
}

// Worker holds the configuration for the worker
type Worker struct {
	Database     Database
	PollInterval int `envconfig:"WORKER_POLL_INTERVAL" default:"1"` // seconds
	TaskTimeout  int `envconfig:"WORKER_TASK_TIMEOUT" default:"30"` // seconds
	Concurrency  int `envconfig:"WORKER_CONCURRENCY" default:"1"`   // number of concurrent workers
}
