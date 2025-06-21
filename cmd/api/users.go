package main

import (
	"net/http"

	"github.com/MisterDodik/Barbershop/internal/store"
)

type userKey string

const userCtx userKey = "user"

func getUserFromContext(r *http.Request) *store.User {
	return r.Context().Value(userCtx).(*store.User)
}

func (app *application) getMyAppointments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := getUserFromContext(r)
	appointments, err := app.store.TimeSlots.GetMyAppointments(ctx, user.ID)

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

	if err := app.jsonResponse(w, http.StatusOK, appointments); err != nil {
		app.internalServerError(w, r, err)
		return
	}

}
