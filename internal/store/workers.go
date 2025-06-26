package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type WorkerProfile struct {
	UserID              int64             `json:"user_id"`
	WorkingHours        map[string]string `json:"working_hours"`
	AppointmentDuration time.Duration     `json:"appointment_duration,string"`
	PauseBetween        time.Duration     `json:"pause_between,string"`
}

type WorkerProfileStorage struct {
	db *sql.DB
}

func (p *WorkerProfileStorage) CreateOrUpdateSettings(ctx context.Context, workerID int64, workingHours map[string]string, appointmentDuration, pauseBetween int) error {
	query := `
		INSERT INTO worker_profile (
			user_id, working_hours, appointment_duration, pause_between
		)
		VALUES ($1, $2::jsonb, $3::INTERVAL, $4::INTERVAL)
		ON CONFLICT (user_id) DO UPDATE SET
			working_hours = EXCLUDED.working_hours,
			appointment_duration = EXCLUDED.appointment_duration,
			pause_between = EXCLUDED.pause_between;
	`
	jsonData, err := json.Marshal(workingHours)
	if err != nil {
		return err
	}
	_, err = p.db.ExecContext(
		ctx,
		query,
		workerID,
		jsonData,
		fmt.Sprintf("%dm", appointmentDuration),
		fmt.Sprintf("%dm", pauseBetween),
	)
	if err != nil {
		return err
	}
	return nil
}

func (p *WorkerProfileStorage) GetSettings(ctx context.Context, workerID int64) (*WorkerProfile, error) {
	query := `
		SELECT 
			user_id, working_hours, appointment_duration, pause_between FROM worker_profile
		WHERE
			user_id = $1
	`
	var (
		rawJSON     []byte
		rawDuration string
		rawPause    string
	)
	var settings WorkerProfile
	err := p.db.QueryRowContext(
		ctx,
		query,
		workerID,
	).Scan(
		&settings.UserID,
		&rawJSON,
		&rawDuration,
		&rawPause,
	)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, Error_NotFound
		default:
			return nil, err
		}
	}

	err = json.Unmarshal(rawJSON, &settings.WorkingHours)
	if err != nil {
		return nil, err
	}
	settings.AppointmentDuration, err = parsePgIntervalToMinutes(rawDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to parse appointment_duration: %w", err)
	}
	settings.PauseBetween, err = parsePgIntervalToMinutes(rawPause)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pause_between: %w", err)
	}

	return &settings, nil
}
func parsePgIntervalToMinutes(pg string) (time.Duration, error) {
	t, err := time.Parse("15:04:05", pg)
	if err != nil {
		return 0, err
	}
	totalMinutes := t.Hour()*60 + t.Minute()
	return time.Duration(totalMinutes) * time.Minute, nil
}
