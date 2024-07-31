package posts

import (
	"errors"
	"fmt"
	"github.com/hashicorp/go-uuid"
	"myredditclone/pkg/session"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrRecordNotFound = errors.New("Current record doesn't exist")
)

var _ PostRepo = NewPostMemoryRepository()

type PostMemoryRepository struct {
	lastID uint64
	data   map[string]Post
	mu     sync.RWMutex
}

func NewPostMemoryRepository() *PostMemoryRepository {
	return &PostMemoryRepository{
		data: map[string]Post{},
	}
}

func MapToSlice[K comparable, V any](m map[K]V) []V {
	s := make([]V, 0, len(m))
	for _, v := range m {
		s = append(s, v)
	}
	return s
}

func (repo *PostMemoryRepository) GetAll() ([]Post, error) {
	return MapToSlice(repo.data), nil
}

func (repo *PostMemoryRepository) GetByID(id string) (Post, error) {
	repo.mu.Lock()
	post, ok := repo.data[id]
	if !ok {
		return Post{}, ErrRecordNotFound
	}
	return post, nil
}

func (repo *PostMemoryRepository) Add(item *Post) (lastID uint64, err error) {
	repo.mu.Lock()
	atomic.AddUint64(&repo.lastID, 1)
	defer repo.mu.Unlock()
	item.ID = strconv.FormatUint(repo.lastID, 10)
	repo.data[item.ID] = *item
	return repo.lastID, nil
}

func (repo *PostMemoryRepository) Update(newPost Post) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	_, ok := repo.data[newPost.ID]
	if !ok {
		return ErrRecordNotFound
	}
	repo.data[newPost.ID] = newPost
	return nil
}

func (repo *PostMemoryRepository) Delete(id string) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	_, ok := repo.data[id]
	if !ok {
		return ErrRecordNotFound
	}
	delete(repo.data, id)
	return nil
}

func (repo *PostMemoryRepository) AddComment(postID string, newCommentBody string, sess session.Session) (Post, error) {
	repo.mu.RLock()
	post, ok := repo.data[postID]
	repo.mu.RUnlock()
	if !ok {
		return Post{}, ErrRecordNotFound
	}
	randomID, err := uuid.GenerateRandomBytes(16)
	if err != nil {
		return Post{}, ErrRecordNotFound
	}
	comm := Comment{
		Created: time.Now().String(),
		Author: Author{
			Username: sess.Login,
			ID:       strconv.FormatUint(sess.UserID, 10),
		},
		Body: newCommentBody,
		ID:   fmt.Sprintf("%x", randomID),
	}
	post.Comments = append(post.Comments, comm)
	repo.mu.Lock()
	repo.data[postID] = post
	repo.mu.Unlock()
	post.Votes = MapToSlice(post.VotesFromDB)
	return post, nil
}

func (repo *PostMemoryRepository) DeleteComment(postID string, commID string, sess session.Session) (Post, error) {
	repo.mu.RLock()
	post, ok := repo.data[postID]
	if !ok {
		return Post{}, ErrRecordNotFound
	}
	for i, com := range post.Comments {
		if com.ID == commID && com.Author.Username == sess.Login && com.Author.ID == strconv.FormatUint(sess.UserID, 10) {
			post.Comments[i] = post.Comments[len(post.Comments)-1]
			post.Comments = post.Comments[:len(post.Comments)-1]
			repo.mu.Lock()
			repo.data[postID] = post
			repo.mu.Unlock()
			post.Votes = MapToSlice(post.VotesFromDB)
			return post, nil
		}
	}
	return Post{}, ErrRecordNotFound
}

func (repo *PostMemoryRepository) Vote(postID, userID string, newVote int8) (Post, error) {
	repo.mu.RLock()
	post, ok := repo.data[postID]
	repo.mu.RUnlock()
	if !ok {
		return Post{}, ErrRecordNotFound
	}
	lastVote, isVoteExist := post.VotesFromDB[userID]
	//удалить старые значения, если существовал ранее
	if isVoteExist {
		post.Score -= int64(lastVote.Vote)
		if lastVote.Vote == 1 {
			post.UpvoteNum--
		}
	}
	if newVote != 0 {
		post.VotesFromDB[userID] = Vote{
			User: userID,
			Vote: newVote,
		}
		post.Score += int64(newVote)
		if newVote == 1 {
			post.UpvoteNum++
		}
	} else {
		if !isVoteExist {
			return Post{}, ErrRecordNotFound
		}
		delete(post.VotesFromDB, userID)
	}

	if len(post.VotesFromDB) != 0 {
		post.UpvotePercentage = uint8(100 * int(post.UpvoteNum) / len(post.VotesFromDB))
	} else {
		post.UpvotePercentage = 0
	}
	repo.mu.Lock()
	repo.data[postID] = post
	repo.mu.Unlock()
	post.Votes = MapToSlice(post.VotesFromDB)
	return post, nil
}
