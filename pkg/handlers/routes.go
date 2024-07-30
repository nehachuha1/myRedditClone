package handlers

import (
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"myredditclone/pkg/middleware"
	"myredditclone/pkg/session"
	"net/http"
)

func GenerateRoutes(uh UserHandler) *mux.Router {
	r := mux.NewRouter()
	r.Handle("/", http.FileServer(http.Dir("/static/html")))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))

	r.HandleFunc("/api/register", uh.Register).Methods("POST")
	r.HandleFunc("/api/login", uh.Login).Methods("POST")
	return r
}

func PostProcess(r *mux.Router, sm *session.SessionsManager, logger *zap.SugaredLogger) http.Handler {
	r.Use(middleware.Auth(sm))
	r.Use(middleware.AccessLog(logger))
	r.Use(middleware.Panic)
	return r
}
