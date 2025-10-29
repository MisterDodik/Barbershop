package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/MisterDodik/Barbershop/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type UserPayload struct {
	FirstName string `json:"first_name" validate:"required,max=100"`
	LastName  string `json:"last_name" validate:"required,max=100"`
	Username  string `json:"username" validate:"required,max=100"`
	Email     string `json:"email" validate:"required,email,max=255"`
	Password  string `json:"password" validate:"required,min=3,max=72"`
	Role      string `json:"role" validate:"omitempty,oneof=customer worker"`
}

func (app *application) registerUserHandler(w http.ResponseWriter, r *http.Request) {
	var payload UserPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user := &store.User{
		FirstName: payload.FirstName,
		LastName:  payload.LastName,
		Username:  payload.Username,
		Email:     payload.Email,
		Role:      "customer",
	}
	if payload.Role == "worker" {
		user.Role = "worker"
	}

	if err := user.Password.Set(payload.Password); err != nil {
		app.internalServerError(w, r, err)
		return
	}
	plainToken := uuid.New()
	invitationExp := time.Hour * 24

	err := app.store.Users.CreateAndInvite(r.Context(), user, plainToken.String(), invitationExp)
	if err != nil {
		switch err {
		case store.Error_DuplicateEmail:
			app.badRequestResponse(w, r, err)
		case store.Error_DuplicateUsername:
			app.badRequestResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}
	isProdEnv := app.config.env == "production"

	activationUrl := fmt.Sprintf("%s/activate?token=%s", app.config.frontEndURL, plainToken)
	log.Printf("Activation token: %s", plainToken.String())
	vars := struct {
		BarbershopName string
		Username       string
		ActivationURL  string
	}{
		BarbershopName: app.config.BarbershopName,
		Username:       payload.Username,
		ActivationURL:  activationUrl,
	}

	statusCode, err := app.mailer.Send("user_invitation.tmpl", payload.Username, payload.Email, vars, isProdEnv)
	if err != nil && statusCode != http.StatusAccepted {
		log.Printf("an error %s occured. Deleting user invitations from the database", err)
		err = app.store.Users.DeleteUserWithInvitation(r.Context(), user.ID, plainToken.String())
		app.internalServerError(w, r, err)
		return
	}
	if err := app.jsonResponse(w, http.StatusCreated, nil); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

func (app *application) activateUserHandler(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")

	if err := app.store.Users.Activate(r.Context(), token); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if err := app.jsonResponse(w, http.StatusAccepted, "user successfully activated"); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

type CreateUserTokenPayload struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=3,max=72"`
}

func (app *application) createTokenHandler(w http.ResponseWriter, r *http.Request) {
	var payload CreateUserTokenPayload
	if err := readJSON(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if err := Validate.Struct(payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	ctx := r.Context()
	user, err := app.store.Users.GetByEmail(ctx, payload.Email)
	if err != nil {
		switch err {
		case store.Error_NotFound:
			app.unauthorizedErrorResponse(w, r, err)
		case store.Error_UserNotVerified:
			app.unauthorizedErrorResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}
	if !user.Password.ComparePasswords(payload.Password) {
		app.unauthorizedErrorResponse(w, r, fmt.Errorf("incorrect password"))
		return
	}

	claims := jwt.MapClaims{
		"sub": user.ID,
		"exp": time.Now().Add(app.config.auth.token.expDate).Unix(),
		"iat": time.Now().Unix(),
		"nbf": time.Now().Unix(),
		"iss": app.config.auth.token.iss,
		"aud": app.config.auth.token.iss,
	}

	token, err := app.authenticator.GenerateToken(claims)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	if err := app.jsonResponse(w, http.StatusCreated, token); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}
