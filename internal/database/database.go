package database

import (
	"database/sql"
	"errors"

	_ "github.com/lib/pq"
	"go4.org/syncutil"
)

var (
	once     syncutil.Once
	database *sql.DB
)

func Get() *sql.DB {
	return database
}

func Connect() error {
	err := once.Do(func() (err error) {
		database, err = sql.Open("postgres", "user=postgres dbname=votestreet sslmode=disable")
		return err
	})

	return err
}

func Close() error {
	if database == nil {
		return nil
	}

	return database.Close()
}

func CreateTables() error {
	if err := createUsersTable(); err != nil {
		return err
	}

	if err := createPollsTable(); err != nil {
		return err
	}

	if err := createVotesTable(); err != nil {
		return err
	}

	return nil
}

func execute(query string) error {
	if _, err := database.Exec(query); err != nil {
		return err
	}

	return nil
}

var ErrDatabaseNotConnected = errors.New("database not connected")
