package handlers

import (
	"encoding/json"
	"go.uber.org/zap"
	"io"
	"myredditclone/pkg/session"
	"myredditclone/pkg/user"
	"net/http"
)

type UserHandler struct {
	Logger   *zap.SugaredLogger
	Sessions *session.SessionsManager
	UserRepo user.UserRepo
}

type LoginData struct {
	Username string
	Password string
}

func (u *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		jsonError(w, http.StatusBadRequest, "unknown payload")
	}

	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		jsonError(w, http.StatusBadRequest, "cant read request body")
	}
	ld := &LoginData{}
	err = json.Unmarshal(body, ld)

	if err != nil {
		jsonError(w, http.StatusBadRequest, "cant unpack payload")
		return
	}

	usr, err := u.UserRepo.Authorize(ld.Username, ld.Password)
	if err != nil {
		if err != nil { // формируем ошибку при регистрации
			authErrResp(w, "username", ld.Username, err)
			return
		}
	}

	sess, err := u.Sessions.Create(w, usr.ID, usr.Login)
	if err != nil {
		http.Error(w, `Session isn't create`+err.Error(), http.StatusInternalServerError)
		return
	}
	u.Logger.Infof("Successfully created session for user with ID %v", sess.UserID)
	token, err := session.CreateNewToken(usr)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp, err := json.Marshal(map[string]interface{}{
		"token":       token,
		"Status code": http.StatusFound,
	})
	CheckMarshalError(w, err, resp)
	u.Logger.Infof("Send token on client for user with id: %v ", sess.UserID)
}

func (u *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		jsonError(w, http.StatusBadRequest, "unknown payload")
	}

	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		jsonError(w, http.StatusBadRequest, "cant read request body")
	}
	ld := &LoginData{}
	err = json.Unmarshal(body, ld)

	if err != nil {
		jsonError(w, http.StatusBadRequest, "cant unpack payload")
		return
	}

	usr, err := u.UserRepo.Register(ld.Username, ld.Password)
	if err != nil {
		if err != nil { // формируем ошибку при регистрации
			authErrResp(w, "username", ld.Username, err)
			return
		}
	}

	sess, err := u.Sessions.Create(w, usr.ID, usr.Login)
	if err != nil {
		http.Error(w, `Session isn't create`+err.Error(), http.StatusInternalServerError)
		return
	}
	u.Logger.Infof("Successfully created session for user with ID %v", sess.UserID)
	token, err := session.CreateNewToken(usr)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp, err := json.Marshal(map[string]interface{}{
		"token":       token,
		"Status code": http.StatusFound,
	})
	CheckMarshalError(w, err, resp)
	u.Logger.Infof("Send token on client for user with id: %v ", sess.UserID)
}

func jsonError(w http.ResponseWriter, status int, msg string) {
	resp, err := json.Marshal(map[string]interface{}{
		"status": status,
		"error":  msg,
	})
	CheckMarshalError(w, err, resp)
}

func CheckMarshalError(w http.ResponseWriter, err error, resp []byte) {
	if err != nil {
		http.Error(w, "Marshaling error", http.StatusBadRequest)
		return
	}
	_, err = w.Write(resp)
	if err != nil {
		http.Error(w, "Writing response err", http.StatusInternalServerError)
		return
	}
}

func authErrResp(w http.ResponseWriter, param string, value string, err error) {
	var (
		resp  []byte
		error error
	)
	w.WriteHeader(http.StatusUnprocessableEntity)
	errors := make([]map[string]string, 0)
	errors = append(errors, map[string]string{
		"location": "body",
		"param":    param,
		"value":    value,
		"msg":      err.Error(),
	})
	resp, error = json.Marshal(map[string][]map[string]string{
		"errors": errors,
	})

	CheckMarshalError(w, error, resp)
}
