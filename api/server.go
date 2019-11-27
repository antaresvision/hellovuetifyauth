package api

import "github.com/antaresvision/hellovuetifyauth/db"

type server struct {
	ds *db.Store
}

func NewServer(store *db.Store) *server {
	return &server{ds:store}
}