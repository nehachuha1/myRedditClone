package main

import (
	"fmt"
	"go.uber.org/zap"
	"myredditclone/pkg/handlers"
	"myredditclone/pkg/session"
	"myredditclone/pkg/user"
	"net/http"
)

func main() {
	userRepo := user.NewUserRepository()
	sm := session.NewSessionManager()
	zapLogger, err := zap.NewProduction()

	if err != nil {
		fmt.Println(err)
	}
	defer func() {
		err := zapLogger.Sync()
		if err != nil {
			fmt.Println(err)
		}
	}()

	logger := zapLogger.Sugar()
	userHandler := handlers.UserHandler{
		Logger:   logger,
		Sessions: sm,
		UserRepo: userRepo,
	}
	addHandlersMux := handlers.GenerateRoutes(userHandler)
	addProcessingRouter := handlers.PostProcess(addHandlersMux, sm, logger)

	addr := ":8080"
	logger.Infow("starting server",
		"type", "START",
		"addr", addr,
	)
	err = http.ListenAndServe(addr, addProcessingRouter)
	if err != nil {
		fmt.Println(err)
	}
}
