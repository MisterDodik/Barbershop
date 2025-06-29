package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

func (app *application) TokenAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		if authHeader == "" {
			app.unauthorizedErrorResponse(w, r, fmt.Errorf("authorization header is missing"))
			return
		}

		authComponents := strings.Split(authHeader, " ")

		if len(authComponents) != 2 || authComponents[0] != "Bearer" {
			app.unauthorizedErrorResponse(w, r, fmt.Errorf("authrizaton header malformed"))
			return
		}

		token, err := app.authenticator.ValidateToken(authComponents[1])

		if err != nil {
			app.unauthorizedErrorResponse(w, r, err)
			return
		}

		claims, _ := token.Claims.(jwt.MapClaims)

		userID, err := strconv.ParseInt(fmt.Sprintf("%.f", claims["sub"]), 10, 64)

		if err != nil {
			app.unauthorizedErrorResponse(w, r, err)
			return
		}
		ctx := r.Context()
		user, err := app.store.Users.GetByID(ctx, userID)

		if err != nil {
			app.unauthorizedErrorResponse(w, r, err)
			return
		}

		ctx = context.WithValue(ctx, userCtx, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (app *application) AdminAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		worker := getUserFromContext(r)
		if worker == nil || worker.Role != "worker" {
			app.unauthorizedErrorResponse(w, r, fmt.Errorf("you dont have permissions to access this"))
			return
		}

		next.ServeHTTP(w, r)
	})
}
