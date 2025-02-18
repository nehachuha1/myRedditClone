package user

type User struct {
	ID       uint64
	Login    string
	password string
}

type UserRepo interface {
	Authorize(login, pass string) (User, error)
	Register(login, pass string) (User, error)
}
