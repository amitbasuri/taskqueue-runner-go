package config

import "testing"

func TestDatabase_ToDbConnectionUri(t *testing.T) {
    d := Database{
        Username:     "user",
        Password:     "pass",
        Host:         "localhost",
        Port:         "5432",
        Database:     "tasks",
        SSLMode:      "disable",
        PoolMaxConns: 5,
    }

    got := d.ToDbConnectionUri()
    want := "postgres://user:pass@localhost:5432/tasks?sslmode=disable&pool_max_conns=5"
    if got != want {
        t.Fatalf("ToDbConnectionUri() = %q, want %q", got, want)
    }
}

func TestDatabase_ToMigrationUri(t *testing.T) {
    d := Database{
        Username: "user",
        Password: "pass",
        Host:     "localhost",
        Port:     "5432",
        Database: "tasks",
        SSLMode:  "require",
    }

    got := d.ToMigrationUri()
    want := "pgx5://user:pass@localhost:5432/tasks?sslmode=require"
    if got != want {
        t.Fatalf("ToMigrationUri() = %q, want %q", got, want)
    }
}
