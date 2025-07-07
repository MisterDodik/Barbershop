package main

import (
	"fmt"
	"net/http"

	"github.com/MisterDodik/Barbershop/internal/store"
)

type userKey string

const userCtx userKey = "user"

func getUserFromContext(r *http.Request) *store.User {
	return r.Context().Value(userCtx).(*store.User)
}

func (app *application) getMyInfo(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r)

	if err := app.jsonResponse(w, http.StatusOK, user); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

type ResetPasswordPayload struct {
	NewPassword string `json:"new_password" validate:"required,min=3,max=72"`
}

func (app *application) resetPassword(w http.ResponseWriter, r *http.Request) {
	var payload ResetPasswordPayload

	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	ctx := r.Context()
	user := getUserFromContext(r)

	if err := user.Password.Set(payload.NewPassword); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	err := app.store.Users.ResetPassword(ctx, user)
	if err != nil {
		switch err {
		case store.Error_NotFound:
			app.notFoundResponse(w, r, fmt.Errorf("no changes were made"))
			return
		default:
			app.internalServerError(w, r, err)
			return
		}
	}

	if err := app.jsonResponse(w, http.StatusOK, "password updated"); err != nil {
		app.internalServerError(w, r, err)
		return
	}
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
