package postgres

import (
	"context"
	"database/sql"
	"fmt"

	// Initialize pq driver
	_ "github.com/lib/pq"
)

type Client struct {
	*sql.DB
}

func NewClient(host, username, password, database, sslMode string) (*Client, error) {
	connectionString := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=%s", host, username, password, database, sslMode)

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}
	return &Client{db}, nil
}

func (pg *Client) ListDatabases(ctx context.Context) ([]string, error) {
	stmt := "SELECT datname FROM pg_database;"
	rows, err := pg.QueryContext(ctx, stmt)
	if err != nil {
		return nil, err
	}

	var dbList []string
	for rows.Next() {
		var db string
		err = rows.Scan(&db)
		if err != nil {
			break
		}
		dbList = append(dbList, db)
	}
	return dbList, nil
}
