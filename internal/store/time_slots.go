package store

import (
	"context"
	"database/sql"
	"time"
)

type TimeSlotsStorage struct {
	db *sql.DB
}
type TimeSlot struct {
	ID        int64  `json:"id"`
	IsBooked  bool   `json:"is_booked"`
	StartTime string `json:"start_time"`
	User      User   `json:"user"` //koristicu kad budem fetchovao zakazane termine za korisnika
}

func (s *TimeSlotsStorage) GetFreeSlots(ctx context.Context, selectedDay time.Time) ([]TimeSlot, error) {
	year, month, day := selectedDay.Date()
	start := time.Date(year, month, day, 0, 0, 0, 0, selectedDay.Location())
	end := start.AddDate(0, 0, 1)

	query :=
		`
			SELECT id, is_booked, start_time FROM time_slots 
			WHERE is_booked = FALSE AND
			start_time >= $1::timestamp AND start_time < $2::timestamp;
		`
	rows, err := s.db.QueryContext(
		ctx,
		query,
		start,
		end,
	)

	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, Error_NotFound
		default:
			return nil, err
		}
	}
	var timeSlots []TimeSlot
	for rows.Next() {
		var slot TimeSlot
		err := rows.Scan(
			&slot.ID,
			&slot.IsBooked,
			&slot.StartTime,
		)
		if err != nil {
			return timeSlots, err
		}
		timeSlots = append(timeSlots, slot)
	}

	defer rows.Close()
	return timeSlots, nil
}
func (s *TimeSlotsStorage) GetMyAppointments(ctx context.Context, userID int64) ([]TimeSlot, error) {
	query := `
		SELECT t.id, t.is_booked, t.start_time, u.id, u.username, u.email, u.first_name, u.last_name, u.created_at
		FROM time_slots t
		JOIN users u ON t.user_id = u.id
		WHERE t.user_id = $1
		ORDER BY t.start_time ASC
		LIMIT 10
	`
	rows, err := s.db.QueryContext(
		ctx,
		query,
		userID,
	)

	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, Error_NotFound
		default:
			return nil, err
		}
	}
	var timeSlots []TimeSlot
	for rows.Next() {
		var slot TimeSlot
		err := rows.Scan(
			&slot.ID,
			&slot.IsBooked,
			&slot.StartTime,
			&slot.User.ID,
			&slot.User.Username,
			&slot.User.Email,
			&slot.User.FirstName,
			&slot.User.LastName,
			&slot.User.Created_at,
		)
		if err != nil {
			return timeSlots, err
		}
		timeSlots = append(timeSlots, slot)
	}

	defer rows.Close()
	return timeSlots, nil

}
func (s *TimeSlotsStorage) Book(ctx context.Context, slotID int64, userID int64) error {
	query := `
		UPDATE time_slots
		SET is_booked = true, user_id = $2
		WHERE id = $1 AND is_booked = false
	`
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()
	//TODO  -dodaj da izbaci gresku ako se bukira termin koji je vec zauzet
	rows, err := s.db.ExecContext(
		ctx,
		query,
		slotID,
		userID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := rows.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return Error_NotFound
	}
	return nil
}
