package session

import (
	"github.com/dgrijalva/jwt-go"
	"myredditclone/pkg/user"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

var Key = []byte("osfhvjfblkvbke")

type SessionsManager struct {
	data map[string]*Session
	mu   sync.RWMutex
}

func NewSessionManager() *SessionsManager {
	return &SessionsManager{
		data: make(map[string]*Session, 100),
	}
}

func CreateNewToken(user user.User) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user": map[string]interface{}{
			"username": user.Login,
			"id":       strconv.FormatUint(user.ID, 10),
		},
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(30 * 24 * time.Hour).Unix(),
	})
	return token.SignedString(Key)
}

func (sm *SessionsManager) Check(w http.ResponseWriter, r *http.Request) (*Session, error) {
	token := r.Header.Get("Authorization")
	_, tokenString, ok := strings.Cut(token, "Bearer")
	if !ok {
		return nil, ErrNoAuth
	}
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, ErrNoAuth
		}
		return Key, nil
	})
	if err != nil {
		return nil, ErrNoAuth
	}
	user, ok := (claims["user"]).(map[string]interface{})
	if !ok {
		return nil, ErrNoAuth
	}
	userID, success := (user["id"]).(string)
	if !success {
		return nil, ErrNoAuth
	}
	sm.mu.RLock()
	sess, ok := sm.data[userID]
	sm.mu.RUnlock()
	if !ok {
		return nil, ErrNoAuth
	}
	return sess, nil
}

func (sm *SessionsManager) Create(w http.ResponseWriter, userID uint64, login string) (*Session, error) {
	sess := NewSession(userID, login)
	sm.mu.Lock()
	sm.data[strconv.FormatUint(userID, 10)] = sess
	sm.mu.Unlock()
	return sess, nil
}
