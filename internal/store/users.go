package store

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	Error_DuplicateEmail    = errors.New("a user with that email already exists")
	Error_DuplicateUsername = errors.New("a user with that username already exists")
	Error_UserNotVerified   = errors.New("user has not verified their email")
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
	IsActive   bool     `json:"is_active"`
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

func (u *UserStorage) Create(ctx context.Context, tx *sql.Tx, user *User) error {
	query :=
		`
		INSERT INTO users (username, first_name, last_name,  email, password, roles) 
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at
	`
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()
	err := tx.QueryRowContext(
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
		SELECT id, email, first_name, last_name, username, password, created_at, roles, is_active FROM users 
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
		&user.IsActive,
	)

	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, Error_NotFound
		default:
			return nil, err
		}
	}
	if !user.IsActive {
		return nil, Error_UserNotVerified
	}
	return &user, nil
}

func (u *UserStorage) GetByID(ctx context.Context, userID int64) (*User, error) {
	query := `
		SELECT id, email, first_name, last_name, username, password, created_at, roles, is_active FROM users 
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
		&user.IsActive,
	)

	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, Error_NotFound
		default:
			return nil, err
		}
	}
	if !user.IsActive {
		return nil, Error_UserNotVerified
	}
	return &user, nil
}

func (u *UserStorage) CreateAndInvite(ctx context.Context, user *User, token string, invitationExp time.Duration) error {
	return withTx(u.db, ctx, func(tx *sql.Tx) error {
		//create user
		if err := u.Create(ctx, tx, user); err != nil {
			return err
		}
		log.Printf("created")
		//create invitation
		if err := u.createUserInvitation(ctx, tx, token, invitationExp, user.ID); err != nil {
			return err
		}
		log.Printf("created inv")

		return nil
	})
}

func (u *UserStorage) createUserInvitation(ctx context.Context, tx *sql.Tx, token string, invitationExp time.Duration, userID int64) error {
	query := `
		INSERT INTO user_invitations (token, user_id, expires_at) VALUES ($1, $2, $3)
	`
	_, err := tx.ExecContext(
		ctx,
		query,
		token,
		userID,
		time.Now().Add(invitationExp),
	)

	if err != nil {
		return err
	}
	return nil
}

func (u *UserStorage) Activate(ctx context.Context, token string) error {
	return withTx(u.db, ctx, func(tx *sql.Tx) error {
		user, err := getUserByInvitation(ctx, tx, token)
		if err != nil {
			return err
		}

		user.IsActive = true
		if err := updateUserStatus(ctx, tx, user); err != nil {
			return err
		}

		if err := deleteUserInvitation(ctx, tx, token); err != nil {
			return err
		}
		return nil
	})
}
func getUserByInvitation(ctx context.Context, tx *sql.Tx, token string) (*User, error) {
	query := `
		SELECT u.id, u.email, u.username, u.created_at, u.is_active 
		FROM users u
		JOIN user_invitations i ON i.user_id = u.id
		WHERE i.token = $1 AND i.expires_at > $2
	`
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	var user User
	err := tx.QueryRowContext(
		ctx,
		query,
		token,
		time.Now(),
	).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.Created_at,
		&user.IsActive,
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
func updateUserStatus(ctx context.Context, tx *sql.Tx, user *User) error {
	query := `
		UPDATE users SET is_active = $1
		WHERE id = $2
	`

	_, err := tx.ExecContext(
		ctx,
		query,
		user.IsActive,
		user.ID,
	)

	if err != nil {
		return err
	}
	return nil
}

func deleteUserInvitation(ctx context.Context, tx *sql.Tx, token string) error {
	query := `
		DELETE FROM user_invitations WHERE token = $1
	`

	_, err := tx.ExecContext(
		ctx,
		query,
		token,
	)

	if err != nil {
		return err
	}
	return nil
}

func (u *UserStorage) DeleteUserWithInvitation(ctx context.Context, userID int64, token string) error {
	return withTx(u.db, ctx, func(tx *sql.Tx) error {
		if err := deleteUserInvitation(ctx, tx, token); err != nil {
			return err
		}

		if err := deleteUser(ctx, tx, userID); err != nil {
			return err
		}

		return nil
	})
}

func deleteUser(ctx context.Context, tx *sql.Tx, userID int64) error {
	query := `
		DELETE FROM users
		WHERE id = $1
	`
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	_, err := tx.ExecContext(ctx, query, userID)

	if err != nil {
		return err
	}
	return nil
}
