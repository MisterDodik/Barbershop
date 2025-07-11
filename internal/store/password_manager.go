package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"log"
	"time"
)

var (
	Error_TableNotUpdated = errors.New("table not updated")
	Error_Expired         = errors.New("token expired")
	Error_SamePassword    = errors.New("your new password can't be the same as your old password")
)

type ResetPasswordRequestPayload struct {
	ID        string
	UserID    int64
	ExpiresAt time.Time
}

type PasswordManagerStorage struct {
	db *sql.DB
}

func (u *PasswordManagerStorage) CreateResetPasswordRequest(ctx context.Context, userID int64, hashToken string, expiration time.Duration) error {
	query := `
		INSERT INTO reset_password_requests (id, user_id, expires_at)
		VALUES ($1, $2, $3);
	`
	expiresAt := time.Now().Add(expiration)

	rows, err := u.db.ExecContext(
		ctx,
		query,
		hashToken,
		userID,
		expiresAt,
	)

	if err != nil {
		return err
	}

	n, _ := rows.RowsAffected()
	if n == 0 {
		return Error_TableNotUpdated
	}
	return nil
}
func (u *PasswordManagerStorage) DeleteResetPasswordRequest(ctx context.Context, userID int64) error {
	query := `
		DELETE FROM reset_password_requests
		WHERE user_id = $1;
	`
	rows, err := u.db.ExecContext(
		ctx,
		query,
		userID,
	)

	if err != nil {
		return err
	}

	n, _ := rows.RowsAffected()
	if n == 0 {
		return Error_TableNotUpdated
	}
	return nil
}
func (u *PasswordManagerStorage) UpdatePassword(ctx context.Context, newPassword password, plainToken string) (*int64, error) {
	userID, expTime, passwordHash := u.getValidUserFromToken(ctx, plainToken)

	if userID == nil || expTime == nil || passwordHash == nil {
		return nil, Error_NotFound
	}
	if time.Now().Compare(*expTime) >= 0 {
		if err := u.DeleteResetPasswordRequest(ctx, *userID); err != nil {
			log.Printf("error deleting the request")
		}
		return nil, Error_Expired
	}

	currentPassword := password{
		hash: *passwordHash,
	}
	if currentPassword.ComparePasswords(*newPassword.plain) {
		if err := u.DeleteResetPasswordRequest(ctx, *userID); err != nil {
			log.Printf("error deleting the request")
		}
		return nil, Error_SamePassword
	}

	query := `
		UPDATE users
		SET password = $1
		WHERE id = $2;
	`

	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	rows, err := u.db.ExecContext(
		ctx,
		query,
		newPassword.hash,
		*userID,
	)

	if err != nil {
		return nil, err
	}

	num, _ := rows.RowsAffected()

	if num == 0 {
		return nil, Error_TableNotUpdated
	}

	return userID, nil
}

func (u *PasswordManagerStorage) getValidUserFromToken(ctx context.Context, plainToken string) (*int64, *time.Time, *[]byte) {
	query := `			
		SELECT r.user_id, r.expires_at, u.password
		FROM reset_password_requests r
		JOIN users u ON u.id = r.user_id
		WHERE r.id = $1;
	`

	var userID int64
	var expTime time.Time
	var passwordHash []byte

	hash := sha256.Sum256([]byte(plainToken))
	hashToken := hex.EncodeToString(hash[:])

	err := u.db.QueryRowContext(
		ctx,
		query,
		hashToken,
	).Scan(
		&userID,
		&expTime,
		&passwordHash,
	)

	if err != nil {
		return nil, nil, nil
	}

	return &userID, &expTime, &passwordHash
}
