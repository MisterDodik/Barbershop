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
	ID              int64  `json:"id"`
	IsBooked        bool   `json:"is_booked"`
	StartTime       string `json:"start_time"`
	User            *User  `json:"user,omitempty"`
	WorkerID        int64  `json:"worker_id"`
	WorkerFirstName string `json:"worker_first_name"`
}
type NumberOfSlots struct {
	StartTime   string `json:"start_time"`
	BookedSlots int    `json:"booked_slots"`
}

func (s *TimeSlotsStorage) GetSlots(ctx context.Context, selectedDay time.Time, WorkerID int64, isBooked bool) ([]TimeSlot, error) {
	year, month, day := selectedDay.Date()
	start := time.Date(year, month, day, 0, 0, 0, 0, selectedDay.Location())
	end := start.AddDate(0, 0, 1)

	query :=
		`
		SELECT 
			t.id, t.is_booked, t.start_time,
			c.id, c.first_name, c.last_name, c.email,
			w.id, w.first_name
		FROM time_slots t
		LEFT JOIN users c ON c.id = t.user_id
		JOIN users w ON w.id = t.worker_id
		WHERE is_booked = $3 AND
			start_time >= $1::timestamp AND start_time < $2::timestamp AND
			worker_id = $4;
		`
	rows, err := s.db.QueryContext(
		ctx,
		query,
		start,
		end,
		isBooked,
		WorkerID,
	)

	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, Error_NotFound
		default:
			return nil, err
		}
	}
	var (
		userID                     sql.NullInt64
		firstName, lastName, email sql.NullString
	)

	var timeSlots []TimeSlot
	for rows.Next() {
		var slot TimeSlot
		err := rows.Scan(
			&slot.ID,
			&slot.IsBooked,
			&slot.StartTime,
			&userID,
			&firstName,
			&lastName,
			&email,
			&slot.WorkerID,
			&slot.WorkerFirstName,
		)
		if err != nil {
			return timeSlots, err
		}

		if userID.Valid {
			slot.User = &User{
				ID:        userID.Int64,
				FirstName: firstName.String,
				LastName:  lastName.String,
				Email:     email.String,
			}
		} else {
			slot.User = nil
		}

		timeSlots = append(timeSlots, slot)
	}

	defer rows.Close()
	return timeSlots, nil
}
func (s *TimeSlotsStorage) GetMyAppointments(ctx context.Context, userID int64) ([]TimeSlot, error) {
	query := `
		SELECT 	
			t.id, t.is_booked, t.start_time, 		
			customer.id AS customer_id,
			customer.username AS customer_username,
			customer.email AS customer_email,
			customer.first_name AS customer_first_name,
			customer.last_name AS customer_last_name,
			customer.created_at AS customer_created_at,
			customer.roles AS customer_roles,

			worker.id AS worker_id,
			worker.first_name AS worker_first_name
		FROM time_slots t
		JOIN users customer ON t.user_id = customer.id
		JOIN users worker ON t.worker_id = worker.id
		WHERE t.user_id = $1 AND t.is_booked = true
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
			&slot.User.Role,
			&slot.WorkerID,
			&slot.WorkerFirstName,
		)
		if err != nil {
			return timeSlots, err
		}
		timeSlots = append(timeSlots, slot)
	}

	defer rows.Close()
	return timeSlots, nil

}

func (s *TimeSlotsStorage) GetBookedNumberForAMonth(ctx context.Context, month int, workerID int64) ([]NumberOfSlots, error) {
	query := `
		SELECT DATE(start_time) as day, COUNT(*) AS booked_slots FROM time_slots 
		WHERE is_booked = TRUE AND
		EXTRACT(MONTH FROM start_time) = $1 AND
		worker_id = $2
		GROUP BY DATE(start_time)
		ORDER BY day;
	`

	rows, err := s.db.QueryContext(
		ctx,
		query,
		month,
		workerID,
	)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, Error_NotFound
		default:
			return nil, err
		}
	}
	var timeSlots []NumberOfSlots

	for rows.Next() {
		var slot NumberOfSlots
		err := rows.Scan(
			&slot.StartTime,
			&slot.BookedSlots,
		)
		if err != nil {
			return timeSlots, err
		}
		timeSlots = append(timeSlots, slot)
	}

	defer rows.Close()
	return timeSlots, nil
}

func (s *TimeSlotsStorage) Book(ctx context.Context, slotID, workerID, userID int64) error {
	query := `
		UPDATE time_slots
		SET is_booked = true, user_id = $2
		WHERE id = $1 AND worker_id = $3 AND is_booked = false
	`
	//TODO mzd u ovom query treba izbaciti ovo worker_id mzd je double checking bez razloga al aj
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()
	//TODO  -dodaj da izbaci gresku ako se bukira termin koji je vec zauzet
	rows, err := s.db.ExecContext(
		ctx,
		query,
		slotID,
		userID,
		workerID,
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
