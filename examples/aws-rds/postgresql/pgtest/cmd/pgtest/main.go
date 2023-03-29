/*
 Copyright 2019 The Kanister Authors.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

      http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"sigs.k8s.io/yaml"

	// Initialize pq driver
	_ "github.com/lib/pq"
)

const (
	pgHostEnv     = "PG_HOST"
	pgDBEnv       = "PG_DBNAME"
	pgUserEnv     = "PG_USER"
	pgPasswordEnv = "PG_PASSWORD"
	pgSSLEnv      = "PG_SSL"

	port = 8080
)

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

type pgDB struct {
	*sql.DB
}

func main() {
	db := newPGDB()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		info, err := getInfo()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		fmt.Fprintf(w, "Host=%s User=%s DbName=%s ", info.host, info.user, info.dbName)
	})

	http.HandleFunc("/reset", func(w http.ResponseWriter, r *http.Request) {
		err := resetDB(db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		fmt.Fprintln(w, "Reset database")
	})
	http.HandleFunc("/insert", func(w http.ResponseWriter, r *http.Request) {
		err := addRow(db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		fmt.Fprintln(w, "Inserted a row")
	})
	http.HandleFunc("/count", func(w http.ResponseWriter, r *http.Request) {
		count, err := countRows(db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		fmt.Fprintf(w, "Table has %d rows", count)
	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

type pgInfo struct {
	host     string
	dbName   string
	user     string
	password string
}

func getInfo() (*pgInfo, error) {
	host, ok := os.LookupEnv(pgHostEnv)
	if !ok {
		return nil, fmt.Errorf("%s environment variable not set", pgHostEnv)
	}

	dbName, ok := os.LookupEnv(pgDBEnv)
	if !ok {
		return nil, fmt.Errorf("%s environment variable not set", pgDBEnv)
	}
	// Parse databases from config data
	var databases []string
	if err := yaml.Unmarshal([]byte(dbName), &databases); err != nil {
		return nil, err
	}
	if databases == nil {
		return nil, fmt.Errorf("Databases are missing from configmap")
	}

	user, ok := os.LookupEnv(pgUserEnv)
	if !ok {
		return nil, fmt.Errorf("%s environment variable not set", pgUserEnv)
	}
	password, ok := os.LookupEnv(pgPasswordEnv)
	if !ok {
		return nil, fmt.Errorf("%s environment variable not set", pgPasswordEnv)
	}
	return &pgInfo{host: host, dbName: databases[0], user: user, password: password}, nil
}

func newPGDB() *pgDB {
	info, err := getInfo()
	if err != nil {
		panic(err)
	}

	// Initialize connection string.
	var connectionString string = fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable", info.host, info.user, info.password, info.dbName)

	// Initialize connection object.
	db, err := sql.Open("postgres", connectionString)
	checkError(err)

	err = db.Ping()
	checkError(err)
	fmt.Println("Successfully created connection to database")

	return &pgDB{db}
}

func resetDB(db *pgDB) error {
	// Drop previous table of same name if one exists.
	_, err := db.Exec("DROP TABLE IF EXISTS inventory;")
	if err != nil {
		return err
	}
	log.Println("Finished dropping table (if existed)")

	// Create table.
	_, err = db.Exec("CREATE TABLE inventory (id serial PRIMARY KEY, name VARCHAR(50));")
	if err != nil {
		return err
	}
	log.Println("Finished creating table")
	return nil
}

func addRow(db *pgDB) error {
	now := time.Now().Format(time.RFC3339Nano)
	stmt := "INSERT INTO inventory (name) VALUES ($1);"
	_, err := db.Exec(stmt, now)
	if err != nil {
		return err
	}
	log.Printf("Inserted a row\n")
	return nil
}

func countRows(db *pgDB) (int, error) {
	stmt := "SELECT COUNT(*) FROM inventory;"
	row := db.QueryRow(stmt)
	var count int
	err := row.Scan(&count)
	if err != nil {
		return 0, err
	}
	log.Printf("Found %d rows\n", count)
	return count, nil
}
