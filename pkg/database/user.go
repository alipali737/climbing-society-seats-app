package database

import (
	"database/sql"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           int
	Username     string
	PasswordHash string
}

func GetUserFromDatabaseByUsername(username string) (*User, error) {
	db, err := sql.Open("sqlite", "./database.db")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := "SELECT id, username, password_hash FROM users WHERE username = ?"
	stmt, err := db.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	row := stmt.QueryRow(username)

	var user User
	err = row.Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return &user, nil
}

func ValidatePassword(password string, passwordHash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if err != nil {
		return false
	}
	return true
}
