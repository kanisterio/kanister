// Copyright 2020 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package postgres

import (
	"context"
	"database/sql"
	"fmt"

	// Initialize pq driver
	_ "github.com/lib/pq"
)

const DefaultConnectDatabase = "postgres"

// Client is postgres client to access postgres instance
type Client struct {
	*sql.DB
}

// NewClient initializes postgres client
func NewClient(host, username, password, database, sslMode string) (*Client, error) {
	connectionString := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=%s", host, username, password, database, sslMode)

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, err
	}
	return &Client{db}, nil
}

// PingDB tests db connection
func (pg Client) PingDB(ctx context.Context) error {
	return pg.Ping()
}

// ListDatabases returns list of databases in postgres
func (pg Client) ListDatabases(ctx context.Context) ([]string, error) {
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
