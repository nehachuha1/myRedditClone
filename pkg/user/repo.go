package user

import (
	"errors"
	"sync"
	"sync/atomic"
)

var (
	ErrExistUser = errors.New("This user already exists")
	ErrNoUser    = errors.New("There's no user")
	ErrBadPass   = errors.New("Wrong password")
)

var _ UserRepo = NewUserRepository()

type UserRepository struct {
	currentFreeID atomic.Uint64
	data          map[string]User
	mu            sync.RWMutex
}

func NewUserRepository() *UserRepository {
	return &UserRepository{
		data: make(map[string]User, 0),
	}
}

func (repo *UserRepository) Authorize(login, pass string) (User, error) {
	repo.mu.RLock()
	usr, ok := repo.data[login]
	repo.mu.RUnlock()
	if !ok {
		return User{}, ErrNoUser
	}
	if usr.password != pass {
		return User{}, ErrBadPass
	}
	return usr, nil
}

func (repo *UserRepository) Register(login, pass string) (User, error) {
	newUser := User{
		ID:       repo.currentFreeID.Load(),
		Login:    login,
		password: pass,
	}
	repo.mu.Lock()
	defer repo.mu.Unlock()
	_, ok := repo.data[login]
	if !ok {
		repo.data[login] = newUser
		repo.currentFreeID.Add(1)
	} else {
		return User{}, ErrExistUser
	}
	return newUser, nil
}
