package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/asaskevich/govalidator"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"io"
	"myredditclone/pkg/posts"
	"myredditclone/pkg/session"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

type PostHandler struct {
	PostsRepo posts.PostRepo
	Logger    *zap.SugaredLogger
}

func MarshalAndWrite(w http.ResponseWriter, data interface{}) {
	resp, err := json.Marshal(data)
	if err != nil {
		http.Error(w, "Marshaling error", http.StatusBadRequest)
		return
	}
	_, err = w.Write(resp)
	if err != nil {
		http.Error(w, "Writing response error", http.StatusInternalServerError)
	}
}

func SortSlicePosts(elems []posts.Post) []posts.Post {
	sort.Slice(elems, func(i, j int) bool {
		if elems[i].Score == elems[j].Score {
			return elems[i].Created < elems[j].Created
		}
		return elems[i].Score > elems[j].Score
	})
	return elems
}

func (ph *PostHandler) List(w http.ResponseWriter, r *http.Request) {
	elems, err := ph.PostsRepo.GetAll()
	if err != nil {
		http.Error(w, "List error: DB err - Get all", http.StatusInternalServerError)
		return
	}
	elems = SortSlicePosts(elems)
	for i := range elems {
		elems[i].Votes = posts.MapToSlice(elems[i].VotesFromDB)
	}
	w.WriteHeader(http.StatusOK)
	MarshalAndWrite(w, elems)
}

func (ph *PostHandler) Validate(post posts.Post) (param, value string, err error) {
	if post.URL != "" && post.Text != "" {
		return "urlAndText", post.URL + post.Text, fmt.Errorf("data was obtained simultaneously with two types of posts - containing links and text")
	}
	if post.URL == "" && post.Text == "" {
		return "noUrlAndNoText", "", fmt.Errorf("it was not specified what type of data came in")
	}
	if post.URL != "" {
		isURL := govalidator.IsURL(post.URL)
		if !isURL {
			return "URL", post.URL, fmt.Errorf("URL is not valid")
		}
	}
	return "", "", nil
}

func (ph *PostHandler) AddDefaultFieldsPost(post *posts.Post, sess *session.Session) {
	post.Score = 1
	post.UpvoteNum = 1
	post.Author.ID = strconv.FormatUint(sess.UserID, 10)
	post.Author.Username = sess.Login
	post.VotesFromDB = make(map[string]posts.Vote)
	post.VotesFromDB[post.Author.ID] = posts.Vote{User: post.Author.ID, Vote: 1}
	post.UpvotePercentage = 100
	post.Created = time.Now().Format("2006-01-02T15:04:05.000")
}

func (ph *PostHandler) Add(w http.ResponseWriter, r *http.Request) {
	post := new(posts.Post)
	bytes, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, `Bad request`, http.StatusBadRequest)
		return
	}
	err = json.Unmarshal(bytes, post)
	if err != nil {
		http.Error(w, `Bad form`, http.StatusBadRequest)
		return
	}
	param, value, err := ph.Validate(*post)
	if err != nil {
		authErrResp(w, param, value, err)
		return
	}
	sess, err := session.SessionFromContext(r.Context())
	if err != nil {
		http.Error(w, `Session error`, http.StatusBadRequest)
		return
	}
	ph.AddDefaultFieldsPost(post, sess)
	lastID, err := ph.PostsRepo.Add(post)
	if err != nil {
		http.Error(w, `Add error: DB err - Add`, http.StatusInternalServerError)
		return
	}
	post.Votes = posts.MapToSlice(post.VotesFromDB)
	w.WriteHeader(http.StatusOK)
	MarshalAndWrite(w, post)
	ph.Logger.Infof("Add new post, LastInsertPostId: %v", lastID)
}

func (ph *PostHandler) ListPost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	postID, ok := vars["POST_ID"]
	if !ok {
		http.Error(w, "Request URL hasn't POST_ID", http.StatusBadRequest)
		return
	}
	post, err := ph.PostsRepo.GetByID(postID)
	if err != nil {
		http.Error(w, `ListPost error: DB err - GetByID`, http.StatusInternalServerError)
		return
	}
	post.Views++
	err = ph.PostsRepo.Update(post)
	if err != nil {
		http.Error(w, `ListPost error: DB err - Update`, http.StatusInternalServerError)
		return
	}
	post.Votes = posts.MapToSlice(post.VotesFromDB)
	w.WriteHeader(http.StatusOK)
	MarshalAndWrite(w, post)
	ph.Logger.Infof("View post with ID: %v", post.ID)
}

func (ph *PostHandler) AddComment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	postID, ok := vars["POST_ID"]
	if !ok {
		http.Error(w, "Request URL hasn't POST_ID", http.StatusBadRequest)
		return
	}
	sess, err := session.SessionFromContext(r.Context())
	if err != nil {
		http.Error(w, "You aren't authorize", http.StatusBadRequest)
		return
	}
	bytes, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, `Bad request`, http.StatusBadRequest)
		return
	}
	comments := make(map[string]string)

	err = json.Unmarshal(bytes, &comments)

	var newComment string
	if newComment, ok = comments["comment"]; !ok {
		http.Error(w, `Comment is absent`, http.StatusBadRequest)
		return
	}

	post, err := ph.PostsRepo.AddComment(postID, newComment, *sess)
	if err != nil {
		http.Error(w, `AddComment error: DB err - AddComment`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	MarshalAndWrite(w, post)
	ph.Logger.Infof("Insert new comment with body: %x, at post with ID: %v", newComment, postID)
}

func (ph *PostHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	requestVars := mux.Vars(r)
	postID, ok := requestVars["POST_ID"]
	if !ok {
		http.Error(w, "Request URL hasn't POST_ID", http.StatusBadRequest)
		return
	}
	commID, ok := requestVars["COMMENT_ID"]
	if !ok {
		http.Error(w, "Request URL hasn't COMMENT_ID", http.StatusBadRequest)
		return
	}
	sess, err := session.SessionFromContext(r.Context())
	if err != nil {
		http.Error(w, "You aren't authorize", http.StatusBadRequest)
		return
	}
	post, err := ph.PostsRepo.DeleteComment(postID, commID, *sess)
	if err != nil {
		http.Error(w, `DeleteComment error: DB err - DeleteComment`, http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
	MarshalAndWrite(w, post)
	ph.Logger.Infof("Delete comment with ID: %v, at post with ID^ %v", commID, postID)
}

func (ph *PostHandler) Vote(w http.ResponseWriter, r *http.Request) {
	_, strVote, _ := strings.Cut(r.URL.Path, "/api/post/")
	_, strVote, _ = strings.Cut(strVote, "/")
	requestVars := mux.Vars(r)
	postID, ok := requestVars["POST_ID"]
	if !ok {
		http.Error(w, "Request URL hasn't POST_ID", http.StatusBadRequest)
		return
	}
	sess, err := session.SessionFromContext(r.Context())
	if err != nil {
		http.Error(w, "You aren't authorize", http.StatusBadRequest)
		return
	}

	userIDStr := strconv.FormatUint(sess.UserID, 10)

	var newVote int8
	switch strVote {
	case "upvote":
		newVote = 1
	case "downvote":
		newVote = -1
	case "unvote":
		newVote = 0
	default:
		http.Error(w, `The vote type wasn't sent`, http.StatusBadRequest)
		return
	}
	post, err := ph.PostsRepo.Vote(postID, userIDStr, newVote)
	if err != nil {
		http.Error(w, `Vote error: DB err - Vote`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	MarshalAndWrite(w, post)
	ph.Logger.Infof("Add new reaction: %v at post with ID: %v for user with ID: %v", strVote, post.ID, sess.UserID)
}

func (ph *PostHandler) GetAllAtTheCategory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	category, ok := vars["CATEGORY_NAME"]
	if !ok {
		http.Error(w, "Request URL hasn't CATEGORY_NAME", http.StatusBadRequest)
		return
	}
	elems, err := ph.PostsRepo.GetAll()
	if err != nil {
		http.Error(w, `GetAllAtTheCategory: DB err - GetAll. `, http.StatusInternalServerError)
		return
	}
	needElems := make([]posts.Post, 0)
	for _, v := range elems {
		if v.Category == category {
			needElems = append(needElems, v)
		}
	}
	for i := range needElems {
		needElems[i].Votes = posts.MapToSlice(needElems[i].VotesFromDB)
	}
	needElems = SortSlicePosts(needElems)
	w.WriteHeader(http.StatusOK)
	MarshalAndWrite(w, needElems)
	ph.Logger.Infof("Viewed all posts at category: %v", category)
}

func (ph *PostHandler) Delete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	postID, ok := vars["POST_ID"]
	if !ok {
		http.Error(w, "Request URL hasn't POST_ID", http.StatusBadRequest)
		return
	}
	post, err := ph.PostsRepo.GetByID(postID)
	if err != nil {
		http.Error(w, `Delete error: DB err - GetByID`, http.StatusInternalServerError)
		return
	}
	sess, err := session.SessionFromContext(r.Context())
	if err != nil {
		http.Error(w, "Session err", http.StatusBadRequest)
		return
	}
	if strconv.FormatUint(sess.UserID, 10) != post.Author.ID {
		http.Error(w, "The post was not deleted by its creator", http.StatusBadRequest)
		return
	}
	err = ph.PostsRepo.Delete(postID)
	if err != nil {
		http.Error(w, `Delete error: DB err - Delete`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-type", "application/json")
	respJSON, err := json.Marshal(struct {
		Message string `json:"message"`
	}{Message: "success"})

	if err != nil {
		http.Error(w, "Marshalling err", http.StatusBadRequest)
		return
	}
	_, err = w.Write(respJSON)
	if err != nil {
		http.Error(w, "Writing response err", http.StatusInternalServerError)
		return
	}

	ph.Logger.Infof("Delete post with ID: %v for his creator-user with ID: %v", postID, sess.UserID)
}

func (ph *PostHandler) GetAllAtUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userLogin, ok := vars["USER_LOGIN"]
	if !ok {
		http.Error(w, "Request URL hasn't USER_LOGIN", http.StatusBadRequest)
		return
	}

	elems, err := ph.PostsRepo.GetAll()
	if err != nil {
		http.Error(w, `GetAllAtUser error: DB err - GetAll`, http.StatusInternalServerError)
		return
	}
	needElems := make([]posts.Post, 0)
	for _, v := range elems {
		if v.Author.Username == userLogin {
			needElems = append(needElems, v)
		}
	}
	for i := range needElems {
		needElems[i].Votes = posts.MapToSlice(needElems[i].VotesFromDB)
	}
	needElems = SortSlicePosts(needElems)
	w.WriteHeader(http.StatusOK)
	MarshalAndWrite(w, needElems)
	ph.Logger.Infof("Viewed all user's posts with Login: %v", userLogin)
}
