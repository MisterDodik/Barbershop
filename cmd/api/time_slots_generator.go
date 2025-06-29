package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/MisterDodik/Barbershop/internal/store"
	"github.com/go-chi/chi/v5"
)

type CustomSlotPayload struct {
	StartTime           string `json:"start_time" validate:"required"`
	AppointmentDuration string `json:"appointment_duration" validate:"required"`
}

func (app *application) RemoveSlot(w http.ResponseWriter, r *http.Request) {
	slotIDstr := chi.URLParam(r, "slotID")
	slotID, err := strconv.ParseInt(slotIDstr, 10, 64)

	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	err = app.store.TimeSlots.RemoveSlot(r.Context(), slotID)
	if err != nil {
		switch err {
		case store.Error_NotFound:
			app.notFoundResponse(w, r, fmt.Errorf("no changes have been made"))
			return
		default:
			app.internalServerError(w, r, err)
			return
		}
	}

	if err := app.jsonResponse(w, http.StatusOK, "slot removed"); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}
func (app *application) AddCustomSlot(w http.ResponseWriter, r *http.Request) {
	var payload CustomSlotPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.internalServerError(w, r, err)
		return
	}
	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	startTime, err := time.Parse(time.DateTime, payload.StartTime)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	duration, err := time.ParseDuration(payload.AppointmentDuration)

	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	worker := getUserFromContext(r)
	workerID := worker.ID

	closestAvailable, err := app.store.TimeSlots.CreateNewSlot(r.Context(), workerID, startTime, duration)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	if closestAvailable != nil {
		app.conflictResponse(w, r, fmt.Errorf("unable to add a new slot, but it would overlap with an existing one"))
		return
	}

	if err := app.jsonResponse(w, http.StatusCreated, "database updated"); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) GenerateSlots(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	worker := getUserFromContext(r)
	workerID := worker.ID

	settings, err := app.store.Workers.GetSettings(ctx, workerID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	daysToGenerateString := chi.URLParam(r, "daysCount")
	daysToGenerate, err := strconv.Atoi(daysToGenerateString)
	if err != nil {
		daysToGenerate = 7
	}

	err = app.parseWorkingHours(ctx, workerID, settings.WorkingHours, settings.AppointmentDuration, settings.PauseBetween, daysToGenerate)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.jsonResponse(w, http.StatusOK, "successfully created slots"); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) parseWorkingHours(ctx context.Context, workerID int64, workingHours map[string]string, duration, pause time.Duration, daysAhead int) error {
	days := []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}

	startingIndex := getTodayIndex(days)

	currentDate := time.Now()
	for i := 0; i < daysAhead; i++ {
		currentDay := days[startingIndex]
		startingIndex = (startingIndex + 1) % len(days)

		timeRange, ok := workingHours[currentDay] //radni sati
		log.Print(ok, workingHours[currentDay], currentDay)
		if !ok {
			currentDate = currentDate.AddDate(0, 0, 1)
			i--
			continue
		}

		startTime, endTime, err := parseTimeRange(timeRange)
		if err != nil {
			log.Printf("invalid time range %q for %s: %v", timeRange, currentDay, err)
			continue
		}

		//"2025-07-02 09:30:00"
		for timeOnlyLessOrEqual(startTime, endTime) {
			appointment, err := time.Parse(time.DateTime, fmt.Sprintf("%v %v", currentDate.Format(time.DateOnly), startTime.Format(time.TimeOnly)))
			if err != nil {
				return err
			}

			newTime, err := app.store.TimeSlots.CreateNewSlot(ctx, workerID, appointment, duration)
			if err != nil {
				return err
			}
			if newTime != nil {
				//conflict: try next available time
				startTime = newTime.Add(pause)
				continue
			}
			startTime = startTime.Add(duration + pause)
		}
		currentDate = currentDate.AddDate(0, 0, 1)
	}
	return nil
}
func getTodayIndex(days []string) int {
	todayString := strings.ToLower(time.Now().Weekday().String())
	for i, day := range days {
		if day == todayString {
			return i
		}
	}
	return -1
}

func parseTimeRange(timeRange string) (startTime, endTime time.Time, err error) {
	parts := strings.Split(timeRange, "-")
	if len(parts) != 2 {
		return startTime, endTime, fmt.Errorf("expected 2 parts but got %d", len(parts))
	}

	startTime, err = time.Parse("15:04", parts[0])
	if err != nil {
		return
	}
	endTime, err = time.Parse("15:04", parts[1])
	return
}

func timeOnlyLessOrEqual(t1, t2 time.Time) bool {
	h1, m1, s1 := t1.Clock()
	h2, m2, s2 := t2.Clock()

	if h1 < h2 {
		return true
	}
	if h1 > h2 {
		return false
	}
	if m1 < m2 {
		return true
	}
	if m1 > m2 {
		return false
	}
	return s1 <= s2
}
