package main

import (
	"context"
	"fmt"
	"net/url"

	"github.com/jackc/pgx/v5"
)

const (
	defaultHost = "localhost"
	defaultPort = 5432
)

// dbSetup holds the Postgres client and credentials (getDbClientAndCredentials).
type dbSetup struct {
	conn             *pgx.Conn
	user, password   string
	host             string
	port             int
	dbName           string
	connectionString string
}

func (d *dbSetup) close() {
	if d != nil && d.conn != nil {
		_ = d.conn.Close(context.Background())
		d.conn = nil
	}
}

func postgresClient(ctx context.Context, user, password, host string, port int, database string) (*pgx.Conn, error) {
	u := url.URL{Scheme: "postgres", Host: fmt.Sprintf("%s:%d", host, port), Path: "/" + database}
	if user != "" {
		if password != "" {
			u.User = url.UserPassword(user, password)
		} else {
			u.User = url.User(user)
		}
	}
	return pgx.Connect(ctx, u.String())
}

// formatConnectionString mirrors format-connection-string.ts.
func formatConnectionString(user, password, host, db string) string {
	conn := "postgres://"
	if user != "" {
		conn += url.QueryEscape(user)
	}
	if password != "" {
		conn += ":" + url.QueryEscape(password)
	}
	if user != "" || password != "" {
		conn += "@"
	}
	return conn + host + "/" + db
}

func doesDbExist(ctx context.Context, conn *pgx.Conn, dbName string) bool {
	var name string
	err := conn.QueryRow(ctx, `SELECT datname FROM pg_catalog.pg_database WHERE datname=$1`, dbName).Scan(&name)
	return err == nil
}

func getDbClientAndCredentials(ctx context.Context, dbName, dbURL string, verbose bool) (*dbSetup, error) {
	if dbURL != "" {
		conn, err := pgx.Connect(ctx, dbURL)
		if err != nil {
			if verbose {
				logMessage(fmt.Sprintf("The following error occurred when connecting to the database: %v", err), "verbose")
			}
			return nil, fmt.Errorf("Couldn't connect to PostgreSQL using the database URL you passed. Make sure it's correct and try again.")
		}
		return &dbSetup{conn: conn, connectionString: dbURL}, nil
	}

	d := &dbSetup{user: "postgres", host: defaultHost, port: defaultPort, dbName: dbName}
	conn, err := postgresClient(ctx, d.user, "", d.host, d.port, "postgres")
	if err != nil {
		if verbose {
			logMessage(fmt.Sprintf("The following error occurred when connecting to the database: %v", err), "verbose")
		}
		logMessage("\nCouldn't connect to PostgreSQL with the default credentials (user \"postgres\" without a password).\nMedusa needs to connect to your PostgreSQL server to create the database of your project, so please enter your PostgreSQL credentials below.\n", "info")
		d.user = promptRequired("Enter your Postgres username", "postgres")
		d.password = promptPassword("Enter your Postgres password")
		d.dbName = promptRequired("Enter the name of the database to create for your Medusa project", dbName)
		conn, err = postgresClient(ctx, d.user, d.password, d.host, d.port, "postgres")
		if err != nil {
			conn, err = postgresClient(ctx, d.user, d.password, d.host, d.port, "")
		}
		if err != nil {
			return nil, fmt.Errorf("Couldn't connect to PostgreSQL because of the following error: %v.\n\nMake sure you have PostgreSQL installed and the credentials you provided are correct.\n\nIf you keep running into this issue despite having PostgreSQL installed, please check out the troubleshooting guides: https://docs.medusajs.com/resources/troubleshooting/database-errors", err)
		}
	}
	d.conn = conn
	for doesDbExist(ctx, conn, d.dbName) {
		existing := d.dbName
		next := promptRequired(fmt.Sprintf("A database already exists with the name %s, please enter a name for the database:", existing), existing)
		if next != existing {
			d.dbName = next
		}
	}
	d.connectionString = formatConnectionString(d.user, d.password, d.host, d.dbName)
	return d, nil
}

// runCreateDb mirrors runCreateDb: the database is created and the client
// reconnects to it.
func runCreateDb(ctx context.Context, d *dbSetup) error {
	if _, err := d.conn.Exec(ctx, fmt.Sprintf(`CREATE DATABASE "%s"`, d.dbName)); err != nil {
		return err
	}
	_ = d.conn.Close(ctx)
	conn, err := postgresClient(ctx, d.user, d.password, d.host, d.port, d.dbName)
	if err != nil {
		return err
	}
	d.conn = conn
	return nil
}
