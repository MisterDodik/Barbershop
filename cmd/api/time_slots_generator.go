package main

import (
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

	app.parseWorkingHours(w, r, workerID, settings.WorkingHours, settings.AppointmentDuration, settings.PauseBetween, daysToGenerate)

	if err := app.jsonResponse(w, http.StatusOK, "successfully created slots"); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) parseWorkingHours(w http.ResponseWriter, r *http.Request, workerID int64, data map[string]string, duration, pause time.Duration, daysAhead int) {
	days := []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}
	todayDay := time.Now().Weekday()

	startingIndex := 0
	todayString := strings.ToLower(todayDay.String())
	for i, day := range days {
		if day == todayString {
			startingIndex = i
			break
		}
	}

	currentDate := time.Now()
	for i := 0; i < daysAhead; i++ {
		currentDay := days[startingIndex]

		timeRange, ok := data[currentDay] //radni sati
		startingIndex = (startingIndex + 1) % len(days)
		if !ok {
			i--
			continue
		}
		splitTimes := strings.Split(timeRange, "-") // pocetak i kraj radnog vremena

		if len(splitTimes) != 2 {
			log.Printf("splitTimes %v has %v elements. Expected 2", splitTimes, len(splitTimes))
			currentDate = currentDate.AddDate(0, 0, 1)
			continue
		}
		startTime, err := time.Parse("15:04", splitTimes[0])
		if err != nil {
			log.Printf("cant convert startTime %v to time.Time", splitTimes[0])
			return
		}
		endTime, err := time.Parse("15:04", splitTimes[1])
		if err != nil {
			log.Printf("cant convert endTime %v to time.Time", splitTimes[1])
			return
		}

		//"2025-07-02 09:30:00"
		for timeOnlyLessOrEqual(startTime, endTime) {
			appointment, err := time.Parse(time.DateTime, fmt.Sprintf("%v %v", currentDate.Format(time.DateOnly), startTime.Format(time.TimeOnly)))
			if err != nil {
				fmt.Print(err)
				break
			}

			newTime, err := app.store.TimeSlots.CreateNewSlot(r.Context(), workerID, appointment, duration)
			if err != nil {
				app.internalServerError(w, r, err)
				return
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
