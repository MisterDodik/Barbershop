package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var (
	Error_NotFound       = errors.New("record not found")
	Error_Conflict       = errors.New("resource already exists")
	QueryTimeoutDuration = time.Second * 5
)

type Storage struct {
	Users interface {
		Create(context.Context, *sql.Tx, *User) error
		CreateAndInvite(context.Context, *User, string, time.Duration) error
		DeleteUserWithInvitation(context.Context, int64, string) error
		Activate(context.Context, string) error
		GetByID(context.Context, int64) (*User, error)
		GetByEmail(context.Context, string) (*User, error)
	}
	TimeSlots interface {
		GetSlots(context.Context, time.Time, int64, bool) ([]TimeSlot, error)
		GetMyAppointments(context.Context, int64) ([]TimeSlot, error)
		GetBookedNumberForAMonth(context.Context, int, int64) ([]NumberOfSlots, error)
		Book(context.Context, int64, int64, int64) (*time.Time, error)
		CreateNewSlot(context.Context, int64, time.Time, time.Duration) (*time.Time, error)
		RemoveSlot(context.Context, int64) error
		UpdateStatus(context.Context, int64, string, *int64, string) error
	}
	Workers interface {
		CreateOrUpdateSettings(context.Context, int64, map[string]string, int, int) error
		GetSettings(context.Context, int64) (*WorkerProfile, error)
	}
	PasswordManager interface {
		CreateResetPasswordRequest(context.Context, int64, string, time.Duration) error
		DeleteResetPasswordRequest(context.Context, int64) error
		UpdatePassword(context.Context, password, string) (*int64, error)
	}
}

func NewStorage(db *sql.DB) Storage {
	return Storage{
		Users:           &UserStorage{db},
		TimeSlots:       &TimeSlotsStorage{db},
		Workers:         &WorkerProfileStorage{db},
		PasswordManager: &PasswordManagerStorage{db},
	}
}

func withTx(db *sql.DB, ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)

	if err != nil {
		return nil
	}

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}
