package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

func (app *application) GenerateSlots(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	worker := getUserFromContext(r)
	if worker == nil || worker.Role != "worker" {
		app.unauthorizedErrorResponse(w, r, fmt.Errorf("you dont have permissions to access this"))
		return
	}
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
