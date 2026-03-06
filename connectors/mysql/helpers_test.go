package mysql

import (
	"database/sql"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// validCreds returns a Credentials value with a valid DSN for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"dsn": "user:pass@tcp(localhost:3306)/testdb",
	})
}

// newTestConnector creates a MySQLConnector with a sqlmock database.
// Returns the connector, mock, and a cleanup function.
func newTestConnector() (*MySQLConnector, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		panic("failed to create sqlmock: " + err.Error())
	}

	// Expect the ping and SET statement on every connection.
	mock.ExpectPing()
	mock.ExpectExec("SET SESSION max_execution_time").WillReturnResult(sqlmock.NewResult(0, 0))

	conn := &MySQLConnector{
		timeout:  defaultTimeout,
		rowLimit: defaultRowLimit,
		openDB: func(dsn string) (*sql.DB, error) {
			return db, nil
		},
	}

	return conn, mock, func() { db.Close() }
}
