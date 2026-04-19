package sqlite

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3" // sqlite deriver
)

func NewConnection(dbPath string) *sql.DB {
	db, err := sql.Open("sqlite3", dbPath)

	if err != nil {
		log.Fatal("failed to connect database:", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal("database unreachable:", err)
	}

	return db
}
