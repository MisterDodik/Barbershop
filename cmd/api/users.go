package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"

	"github.com/MisterDodik/Barbershop/internal/store"
	"github.com/google/uuid"
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

type NewPasswordPayload struct {
	NewPassword string `json:"new_password" validate:"required,min=3,max=72"`
	Token       string `json:"token" validate:"required"`
}

func (app *application) updatePassword(w http.ResponseWriter, r *http.Request) {
	var payload NewPasswordPayload

	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user := store.User{}

	if err := user.Password.Set(payload.NewPassword); err != nil {
		app.internalServerError(w, r, err)
		return
	}
	userID, err := app.store.PasswordManager.UpdatePassword(r.Context(), user.Password, payload.Token)
	if err != nil {
		switch err {
		case store.Error_NotFound:
			app.badRequestResponse(w, r, err)
			return
		case store.Error_Expired:
			app.badRequestResponse(w, r, err)
			return
		case store.Error_SamePassword:
			app.badRequestResponse(w, r, err)
			return
		default:
			app.internalServerError(w, r, err)
			return
		}
	}
	if err := app.store.PasswordManager.DeleteResetPasswordRequest(r.Context(), *userID); err != nil {
		app.internalServerError(w, r, err)
		return
	}
	if err := app.jsonResponse(w, http.StatusOK, "password updated"); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

type ResetPasswordRequestPayload struct {
	Email string `json:"email" validate:"required"`
}

func (app *application) requestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var payload ResetPasswordRequestPayload

	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user, err := app.store.Users.GetByEmail(r.Context(), payload.Email) //da provjeri da li postoji korisnik sa tim mejlom
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

	plainToken := uuid.New().String()
	hash := sha256.Sum256([]byte(plainToken))
	hashToken := hex.EncodeToString(hash[:])
	log.Print(plainToken)

	err = app.store.PasswordManager.CreateResetPasswordRequest(r.Context(), user.ID, hashToken, app.config.mail.exp)
	if err != nil {
		switch err {
		case store.Error_TableNotUpdated:
			app.internalServerError(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	resetURL := fmt.Sprintf("%s/reset-password?token=%s", app.config.frontEndURL, plainToken)
	isProdEnv := app.config.env == "production"
	vars := struct {
		Username       string
		ResetURL       string
		BarbershopName string
	}{
		Username:       user.Username,
		ResetURL:       resetURL,
		BarbershopName: app.config.BarbershopName,
	}
	statusCode, err := app.mailer.Send("reset_password.tmpl", user.Username, payload.Email, vars, isProdEnv)
	if err != nil && statusCode != http.StatusAccepted {
		err = app.store.PasswordManager.DeleteResetPasswordRequest(r.Context(), user.ID)
		if err != nil {
			app.internalServerError(w, r, err)
		}
		app.internalServerError(w, r, err)
		return
	}

	if err := app.jsonResponse(w, http.StatusOK, "email sent"); err != nil {
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
