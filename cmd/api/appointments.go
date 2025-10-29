package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/MisterDodik/Barbershop/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type selectedDayPayload struct {
	Day      string `json:"day"`
	WorkerID int64  `json:"worker_id"`
}

// ako se u body json ne stavi nista, onda ce automatski uzeti danasnji dan
func (app *application) getAvailableDates(w http.ResponseWriter, r *http.Request) {
	var selectedDayPayload selectedDayPayload
	if err := readJSON(w, r, &selectedDayPayload); err != nil {
		selectedDayPayload.Day = time.Now().Format(time.DateOnly)
	}

	selectedDay, err := time.Parse(time.DateOnly, selectedDayPayload.Day)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	slots, err := app.store.TimeSlots.GetSlots(r.Context(), selectedDay, selectedDayPayload.WorkerID, false)
	if err != nil {
		switch err {
		case store.Error_NotFound:
			app.notFoundResponse(w, r, err)
			return
		default:
			app.internalServerError(w, r, err)
			return
		}
	}

	if err := app.jsonResponse(w, http.StatusOK, slots); err != nil {
		app.internalServerError(w, r, err)
	}
}

func (app *application) bookAppointment(w http.ResponseWriter, r *http.Request) {
	slotIDstr := chi.URLParam(r, "slotID")
	slotID, err := strconv.ParseInt(slotIDstr, 10, 64)

	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	ctx := r.Context()
	user := getUserFromContext(r)
	var workerID int64
	if user.Role != "worker" {
		workerIDstr := chi.URLParam(r, "workerID")
		workerID, err = strconv.ParseInt(workerIDstr, 10, 64)

		if err != nil {
			app.badRequestResponse(w, r, err)
			return
		}
	} else {
		workerID = user.ID
	}
	workerName, err := app.store.Users.GetByID(ctx, workerID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	slotTime, err := app.store.TimeSlots.Book(ctx, slotID, workerID, user.ID)
	if err != nil {
		switch err {
		case store.Error_NotFound:
			app.notFoundResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}
	if slotTime == nil {
		app.internalServerError(w, r, errors.New("couldn't retrive time to send a mail"))
		return
	}

	bookedDate := slotTime.Format(time.DateOnly)
	bookedTime := slotTime.Format(time.TimeOnly)

	isProdEnv := app.config.env == "production"
	plainToken := uuid.New()

	cancelURL := fmt.Sprintf("%s/cancel?token=%s?id=%s", app.config.frontEndURL, plainToken, slotIDstr)
	log.Print(cancelURL)

	cancelWindow, err := formatDurationFromString(app.config.CancellationWindow)
	if err != nil {
		cancelWindow = app.config.CancellationWindow
	}
	vars := struct {
		BarbershopName  string
		Username        string
		AppointmentDate string
		AppointmentTime string
		BarberName      string
		CancelURL       string
		CancelWindow    string
	}{
		BarbershopName:  app.config.BarbershopName,
		Username:        user.Username,
		AppointmentDate: bookedDate,
		AppointmentTime: bookedTime,
		BarberName:      workerName.Username,
		CancelURL:       cancelURL,
		CancelWindow:    cancelWindow,
	}
	statusCode, err := app.mailer.Send("booked_appointment.tmpl", user.Username, user.Email, vars, isProdEnv)
	if err != nil && statusCode != http.StatusAccepted {
		log.Printf("an error %s occured while sending an email", err)
		app.internalServerError(w, r, err)
		return
	}

	if err := app.jsonResponse(w, http.StatusOK, nil); err != nil {
		app.internalServerError(w, r, err)
	}
}

func formatDurationFromString(s string) (string, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return "", err
	}

	totalMinutes := int(d.Minutes())
	hours := totalMinutes / 60
	minutes := totalMinutes % 60

	if hours > 0 && minutes > 0 {
		return fmt.Sprintf("%dh%dm", hours, minutes), nil
	} else if hours > 0 {
		return fmt.Sprintf("%dh", hours), nil
	}
	return fmt.Sprintf("%dm", minutes), nil
}

func (app *application) cancelAppointment(w http.ResponseWriter, r *http.Request) {
	slotIDstr := chi.URLParam(r, "slotID")
	slotID, err := strconv.ParseInt(slotIDstr, 10, 64)

	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user := getUserFromContext(r)

	if err := app.store.TimeSlots.UpdateStatus(r.Context(), slotID, "available", &user.ID, app.config.CancellationWindow); err != nil {
		switch err {
		case store.Error_NotFound:
			app.notFoundResponse(w, r, err)
			return
		default:
			app.internalServerError(w, r, err)
			return
		}
	}

	if err := app.jsonResponse(w, http.StatusOK, "appointment canceled"); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}
