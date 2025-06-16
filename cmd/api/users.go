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
