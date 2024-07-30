package session

import (
	"context"
	"errors"
)

type sessKey string

var (
	ErrNoAuth          = errors.New("No session found")
	sessionKey sessKey = "session key"
)

type Session struct {
	UserID uint64
	Login  string
}

func NewSession(userID uint64, login string) *Session {
	return &Session{
		UserID: userID,
		Login:  login,
	}
}

func SessionFromContext(ctx context.Context) (*Session, error) {
	sess, ok := ctx.Value(sessionKey).(*Session)
	if !ok || sess == nil {
		return nil, ErrNoAuth
	}
	return sess, nil
}

func ContextWithSession(ctx context.Context, sess *Session) context.Context {
	return context.WithValue(ctx, sessionKey, sess)
}
