package database

import (
	"database/sql"
	"fmt"
	"time"
	_ "github.com/lib/pq"
)

type Database struct {
	db *sql.DB
}

func NewDatabase(dsn string) (*Database, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Database{db: db}, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) CreateUser(name, email string) error {
	query := `INSERT INTO users (name, email) VALUES ($1, $2)`
	_, err := d.db.Exec(query, name, email)
	return err
}

func (d *Database) GetUser(id int) (*User, error) {
	query := `SELECT id, name, email, created_at FROM users WHERE id = $1`
	var user User
	err := d.db.QueryRow(query, id).Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}
