package store

import (
	"context"
	"database/sql"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

var (
	Error_DuplicateEmail    = errors.New("a user with that email already exists")
	Error_DuplicateUsername = errors.New("a user with that username already exists")
)

type User struct {
	ID         int64    `json:"id"`
	Username   string   `json:"username"`
	Email      string   `json:"email"`
	Password   password `json:"-"`
	Created_at string   `json:"created_at"`
}
type UserStorage struct {
	db *sql.DB
}
type password struct {
	plain *string
	hash  []byte
}

func (p *password) Set(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		return err
	}

	p.hash = hash
	p.plain = &password

	return nil
}

func (u *UserStorage) Create(ctx context.Context, user *User) error {
	query :=
		`
		INSERT INTO users (username, email, password) 
		VALUES ($1, $2, $3) RETURNING id, created_at
	`
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()
	err := u.db.QueryRowContext(
		ctx,
		query,
		user.Username,
		user.Email,
		user.Password.hash,
	).Scan(
		&user.ID,
		&user.Created_at,
	)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return Error_DuplicateEmail
		case err.Error() == `pq: duplicate key value violates unique constraint "users_username_key"`:
			return Error_DuplicateUsername
		default:
			return err
		}
	}
	return nil
}
func (u *UserStorage) GetByID(ctx context.Context, userID int64) User {
	var user User
	return user
}
