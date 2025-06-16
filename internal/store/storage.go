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
		Create(context.Context, *User) error
		GetByID(context.Context, int64) User
	}
	TimeSlots interface {
		GetFreeSlots(context.Context, time.Time) ([]TimeSlot, error)
		Book(context.Context, int64, int64) error
	}
}

func NewStorage(db *sql.DB) Storage {
	return Storage{
		Users:     &UserStorage{db},
		TimeSlots: &TimeSlotsStorage{db},
	}
}
