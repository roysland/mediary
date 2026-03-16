package auth

import "net/http"

type User struct {
	ID int64
}

func CurrentUser(r *http.Request) *User {
	// DEV MODE ONLY
	return &User{ID: 1}
}
