// Package pg mirrors @medusajs/utils/src/pg: PostgreSQL helpers used by the
// database commands.
package pg

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// ParseConnectionString mirrors parseConnectionString (pg-connection-string).
func ParseConnectionString(connectionString string) (*pgx.ConnConfig, error) {
	return pgx.ParseConfig(connectionString)
}

// CreateClient mirrors createClient + client.connect.
func CreateClient(ctx context.Context, cfg *pgx.ConnConfig) (*pgx.Conn, error) {
	return pgx.ConnectConfig(ctx, cfg)
}

// CreateDb mirrors createDb.
func CreateDb(ctx context.Context, client *pgx.Conn, databaseName string) error {
	_, err := client.Exec(ctx, `CREATE DATABASE "`+databaseName+`"`)
	return err
}

// DbExists mirrors dbExists.
func DbExists(ctx context.Context, client *pgx.Conn, databaseName string) (bool, error) {
	rows, err := client.Query(ctx, `SELECT datname FROM pg_catalog.pg_database WHERE datname=$1;`, databaseName)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	found := rows.Next()
	return found, rows.Err()
}
