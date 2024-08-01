package handlers

import (
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"myredditclone/pkg/middleware"
	"myredditclone/pkg/session"
	"net/http"
)

func GenerateRoutes(uh UserHandler, ph PostHandler) *mux.Router {
	r := mux.NewRouter()
	r.Handle("/", http.FileServer(http.Dir("/static/html/")))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))

	r.HandleFunc("/api/register", uh.Register).Methods("POST")
	r.HandleFunc("/api/login", uh.Login).Methods("POST")
	r.HandleFunc("/api/posts/", ph.List).Methods("GET")
	r.HandleFunc("/api/posts", ph.Add).Methods("POST")
	r.HandleFunc("/api/posts/{CATEGORY_NAME}", ph.GetAllAtTheCategory).Methods("GET")
	r.HandleFunc("/api/post/{POST_ID}", ph.ListPost).Methods("GET")
	r.HandleFunc("/api/post/{POST_ID}", ph.AddComment).Methods("POST")
	r.HandleFunc("/api/post/{POST_ID}/{COMMENT_ID}", ph.DeleteComment).Methods("DELETE")
	r.HandleFunc("/api/post/{POST_ID}/upvote", ph.Vote).Methods("GET")
	r.HandleFunc("/api/post/{POST_ID}/downvote", ph.Vote).Methods("GET")
	r.HandleFunc("/api/post/{POST_ID}/unvote", ph.Vote).Methods("GET")
	r.HandleFunc("/api/post/{POST_ID}", ph.Delete).Methods("DELETE")
	r.HandleFunc("/api/user/{USER_LOGIN}", ph.GetAllAtUser).Methods("GET")
	r.NotFoundHandler = http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "static/html/index.html")
		})
	return r
}

func PostProcess(r *mux.Router, sm *session.SessionsManager, logger *zap.SugaredLogger) http.Handler {
	r.Use(middleware.Auth(sm))
	r.Use(middleware.AccessLog(logger))
	r.Use(middleware.Panic)
	return r
}
