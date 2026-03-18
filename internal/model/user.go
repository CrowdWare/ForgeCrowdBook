package model

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type User struct {
	ID          int
	Email       string
	DisplayName string
	Bio         string
	Lang        string
	Status      string
	CreatedAt   time.Time
}

func GetUserByEmail(db *sql.DB, email string) (*User, error) {
	row := db.QueryRow(`
		SELECT id, email, display_name, bio, lang, status, created_at
		FROM users
		WHERE email = ?;
	`, email)
	return scanUser(row)
}

func GetUserByID(db *sql.DB, id int) (*User, error) {
	row := db.QueryRow(`
		SELECT id, email, display_name, bio, lang, status, created_at
		FROM users
		WHERE id = ?;
	`, id)
	return scanUser(row)
}

func CreateUser(db *sql.DB, email, displayName string) (*User, error) {
	result, err := db.Exec(`
		INSERT INTO users (email, display_name)
		VALUES (?, ?);
	`, email, displayName)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}

	id64, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("read user id: %w", err)
	}

	return GetUserByID(db, int(id64))
}

func UpdateUserLang(db *sql.DB, id int, lang string) error {
	_, err := db.Exec(`
		UPDATE users
		SET lang = ?
		WHERE id = ?;
	`, lang, id)
	if err != nil {
		return fmt.Errorf("update user language: %w", err)
	}
	return nil
}

func BanUser(db *sql.DB, id int) error {
	_, err := db.Exec(`
		UPDATE users
		SET status = 'banned'
		WHERE id = ?;
	`, id)
	if err != nil {
		return fmt.Errorf("ban user: %w", err)
	}
	return nil
}

func ListUsers(db *sql.DB) ([]User, error) {
	rows, err := db.Query(`
		SELECT id, email, display_name, bio, lang, status, created_at
		FROM users
		ORDER BY created_at DESC;
	`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	users := []User{}
	for rows.Next() {
		var user User
		if err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.DisplayName,
			&user.Bio,
			&user.Lang,
			&user.Status,
			&user.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}
	return users, nil
}

func scanUser(row *sql.Row) (*User, error) {
	var user User
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.DisplayName,
		&user.Bio,
		&user.Lang,
		&user.Status,
		&user.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan user: %w", err)
	}
	return &user, nil
}
