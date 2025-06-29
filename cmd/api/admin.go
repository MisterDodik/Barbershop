package main

import (
	"log"
	"net/http"
	"time"

	"github.com/MisterDodik/Barbershop/internal/store"
)

func (app *application) getCalendarValues(w http.ResponseWriter, r *http.Request) {

	var selectedDayPayload selectedDayPayload
	if err := readJSON(w, r, &selectedDayPayload); err != nil {
		selectedDayPayload.Day = time.Now().Format(time.DateOnly)
	}

	selectedDay, err := time.Parse(time.DateOnly, selectedDayPayload.Day)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	worker := getUserFromContext(r)

	workerID := worker.ID

	_, month, _ := selectedDay.Date()
	data, err := app.store.TimeSlots.GetBookedNumberForAMonth(r.Context(), int(month), workerID)

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

	if err := app.jsonResponse(w, http.StatusOK, data); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

// ako se u body json ne stavi nista, onda ce automatski uzeti danasnji dan
func (app *application) getBookedDates(w http.ResponseWriter, r *http.Request) {
	var selectedDayPayload selectedDayPayload
	if err := readJSON(w, r, &selectedDayPayload); err != nil {
		selectedDayPayload.Day = time.Now().Format(time.DateOnly)
	}

	selectedDay, err := time.Parse(time.DateOnly, selectedDayPayload.Day)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	worker := getUserFromContext(r)

	workerID := worker.ID

	slots, err := app.store.TimeSlots.GetSlots(r.Context(), selectedDay, workerID, true)
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

type WorkSettingsPayload struct {
	WorkingHours        map[string]string `json:"working_hours" validate:"required,dive,keys,required,endkeys"`
	AppointmentDuration int               `json:"appointment_duration" validate:"required,gt=0"`
	PauseBetween        int               `json:"pause_between" validate:"required"`
}

func (app *application) updateWorkSettings(w http.ResponseWriter, r *http.Request) {
	var payload WorkSettingsPayload

	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	worker := getUserFromContext(r)

	workerID := worker.ID

	if err := app.store.Workers.CreateOrUpdateSettings(r.Context(),
		workerID,
		payload.WorkingHours,
		payload.AppointmentDuration,
		payload.PauseBetween); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.jsonResponse(w, http.StatusCreated, "successfully updated worker profile"); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

type WorkerProfileResponse struct {
	UserID              int64             `json:"user_id"`
	WorkingHours        map[string]string `json:"working_hours"`
	AppointmentDuration int               `json:"appointment_duration"`
	PauseBetween        int               `json:"pause_between"`
}

func (app *application) getWorkSettings(w http.ResponseWriter, r *http.Request) {
	worker := getUserFromContext(r)

	workerID := worker.ID

	settings, err := app.store.Workers.GetSettings(r.Context(), workerID)
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
	response := WorkerProfileResponse{
		UserID:              settings.WorkerID,
		WorkingHours:        settings.WorkingHours,
		AppointmentDuration: int(settings.AppointmentDuration.Minutes()),
		PauseBetween:        int(settings.PauseBetween.Minutes()),
	}
	log.Print(settings.AppointmentDuration.Minutes())

	if err := app.jsonResponse(w, http.StatusOK, response); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}
