package utility

import (
	"database/sql"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type newUser struct {
	Username string `flag:"" short:"u" name:"username" help:"Username for new user"`
	Password string `flag:"" short:"p" name:"password" help:"Password for new user"`
}

func (u *newUser) Run() error {
	if u.Username == "" {
		return fmt.Errorf("username must be specified to create a new user")
	}
	if u.Password == "" {
		return fmt.Errorf("password must be specified to create a new user")
	}

	// Generate secure version of password
	hashedPassword, err := u.generateSecureHash()
	if err != nil {
		return fmt.Errorf("failed to generate secure password hash: %v", err)
	}

	// Add new user
	err = addUserToDB(u.Username, hashedPassword)
	if err != nil {
		return fmt.Errorf("failed to create user in db: %v", err)
	}

	return nil
}

func (u newUser) generateSecureHash() (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("error hashing password: %v", err)
	}

	return string(hashedPassword), nil
}

func addUserToDB(username string, hashedPassword string) error {
	// Create DB connection
	db, err := sql.Open("sqlite", "/home/pi/climbing-society-seats-app/database.db")
	if err != nil {
		return err
	}
	defer db.Close()

	// Check if the user already exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", username).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to execute SELECT statement: %v", err)
	}

	if count > 0 {
		return fmt.Errorf("user already exists")
	}

	// Add user
	stmt, err := db.Prepare("INSERT INTO users (username, password_hash) VALUES (?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare INSERT statement: %v", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(username, hashedPassword)
	if err != nil {
		return fmt.Errorf("failed to execute INSERT statement: %v", err)
	}

	return nil
}
