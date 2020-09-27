package middleware

import (
	"context"
	"github.com/go-chi/chi"
	"log"
	"net/http"
	"strconv"
)

func GetClients(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		room, roomKey := chi.URLParam(r, "room"), chi.URLParam(r, "key")
		userId, _ := strconv.Atoi(chi.URLParam(r, "id"))

		var clients = &Client{
			Room:    &room,
			RoomKey: &roomKey,
			Id:      &userId,
		}

		if clients.Room == nil || clients.RoomKey == nil || clients.Id == nil {
			log.Print("All requests params should be send")
			return
		}

		ctx := context.WithValue(r.Context(), "client", clients)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
