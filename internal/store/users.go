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
	FirstName  string   `json:"first_name"`
	LastName   string   `json:"last_name"`
	Username   string   `json:"username"`
	Email      string   `json:"email"`
	Password   password `json:"-"`
	Created_at string   `json:"created_at"`
	Role       string   `json:"role"`
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

func (p *password) ComparePasswords(password string) bool {
	if err := bcrypt.CompareHashAndPassword(p.hash, []byte(password)); err != nil {
		return false
	}
	return true
}

func (u *UserStorage) Create(ctx context.Context, user *User) error {
	query :=
		`
		INSERT INTO users (username, first_name, last_name,  email, password, roles) 
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at
	`
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()
	err := u.db.QueryRowContext(
		ctx,
		query,
		user.Username,
		user.FirstName,
		user.LastName,
		user.Email,
		user.Password.hash,
		user.Role,
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

func (u *UserStorage) GetByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, email, first_name, last_name, username, password, created_at, roles FROM users 
		WHERE email = $1
	`
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	var user User
	err := u.db.QueryRowContext(
		ctx,
		query,
		email,
	).Scan(
		&user.ID,
		&user.Email,
		&user.FirstName,
		&user.LastName,
		&user.Username,
		&user.Password.hash,
		&user.Created_at,
		&user.Role,
	)

	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, Error_NotFound
		default:
			return nil, err
		}
	}
	return &user, nil
}

func (u *UserStorage) GetByID(ctx context.Context, userID int64) (*User, error) {
	query := `
		SELECT id, email, first_name, last_name, username, password, created_at, roles FROM users 
		WHERE id = $1
	`
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	var user User
	err := u.db.QueryRowContext(
		ctx,
		query,
		userID,
	).Scan(
		&user.ID,
		&user.Email,
		&user.FirstName,
		&user.LastName,
		&user.Username,
		&user.Password.hash,
		&user.Created_at,
		&user.Role,
	)

	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, Error_NotFound
		default:
			return nil, err
		}
	}
	return &user, nil
}
