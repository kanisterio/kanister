// Copyright 2026 The Kanister Authors.
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
	"database/sql/driver"
	"errors"
	"io"
	"testing"

	"gopkg.in/check.v1"
)

func Test(t *testing.T) { check.TestingT(t) }

type ClientSuite struct{}

var _ = check.Suite(&ClientSuite{})

// errQueryFailed and errRowFailed are stub-driver errors used to drive the
// ListDatabases error paths.
var (
	errQueryFailed = errors.New("query failed")
	errRowFailed   = errors.New("row iteration failed")
)

// fakeDriver is a minimal database/sql/driver stub used to exercise
// ListDatabases without a live Postgres. Each Open returns a fresh fakeConn
// configured from the package-level fakeConnConfig set by the test.
type fakeDriver struct{}

// fakeConnConfig configures the behavior of the stub connection.
type fakeConnConfig struct {
	// values are the datname rows returned by QueryContext.
	values []string
	// queryErr, if set, is returned by QueryContext instead of any rows.
	queryErr error
	// rowErr, if set, is returned by Rows.Next after all values have been
	// yielded, simulating a mid-iteration error (e.g. a dropped connection).
	rowErr error
	// rowsClosed records whether Rows.Close was called.
	rowsClosed bool
}

var fakeConn = &fakeConnConfig{}

func (fakeDriver) Open(string) (driver.Conn, error) {
	return &fakeConnImpl{cfg: fakeConn}, nil
}

type fakeConnImpl struct {
	cfg *fakeConnConfig
}

func (c *fakeConnImpl) QueryContext(_ context.Context, _ string, _ []driver.NamedValue) (driver.Rows, error) {
	if c.cfg.queryErr != nil {
		return nil, c.cfg.queryErr
	}
	return &fakeRows{cfg: c.cfg}, nil
}

func (c *fakeConnImpl) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("Prepare not implemented")
}

func (c *fakeConnImpl) Close() error { return nil }

func (c *fakeConnImpl) Begin() (driver.Tx, error) {
	return nil, errors.New("Begin not implemented")
}

type fakeRows struct {
	cfg *fakeConnConfig
	pos int
}

func (r *fakeRows) Columns() []string { return []string{"datname"} }

func (r *fakeRows) Close() error {
	r.cfg.rowsClosed = true
	return nil
}

func (r *fakeRows) Next(dest []driver.Value) error {
	if r.pos >= len(r.cfg.values) {
		if r.cfg.rowErr != nil {
			return r.cfg.rowErr
		}
		return io.EOF
	}
	dest[0] = r.cfg.values[r.pos]
	r.pos++
	return nil
}

func (s *ClientSuite) SetUpSuite(_ *check.C) {
	sql.Register("kanister-postgres-fake", fakeDriver{})
}

func (s *ClientSuite) SetUpTest(_ *check.C) {
	fakeConn = &fakeConnConfig{}
}

func (s *ClientSuite) newClient(c *check.C) *Client {
	db, err := sql.Open("kanister-postgres-fake", "")
	c.Assert(err, check.IsNil)
	return &Client{db}
}

func (s *ClientSuite) TestListDatabasesReturnsAllRows(c *check.C) {
	fakeConn.values = []string{"postgres", "template1", "app"}

	client := s.newClient(c)
	dbList, err := client.ListDatabases(context.Background())

	c.Assert(err, check.IsNil)
	c.Assert(dbList, check.DeepEquals, []string{"postgres", "template1", "app"})
	c.Assert(fakeConn.rowsClosed, check.Equals, true)
}

// TestListDatabasesPropagatesRowError guards against silently truncating the
// database list: a mid-iteration error must surface as an error rather than
// returning a partial list with a nil error.
func (s *ClientSuite) TestListDatabasesPropagatesRowError(c *check.C) {
	fakeConn.values = []string{"postgres"}
	fakeConn.rowErr = errRowFailed

	client := s.newClient(c)
	dbList, err := client.ListDatabases(context.Background())

	c.Assert(err, check.NotNil)
	c.Assert(dbList, check.IsNil)
	c.Assert(fakeConn.rowsClosed, check.Equals, true)
}

func (s *ClientSuite) TestListDatabasesPropagatesQueryError(c *check.C) {
	fakeConn.queryErr = errQueryFailed

	client := s.newClient(c)
	dbList, err := client.ListDatabases(context.Background())

	c.Assert(err, check.NotNil)
	c.Assert(dbList, check.IsNil)
}
