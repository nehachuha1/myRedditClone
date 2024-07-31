package posts

import "myredditclone/pkg/session"

type Author struct {
	Username string `json:"username"`
	ID       string `json:"id"`
}

type Vote struct {
	User string `json:"user"` //UserID
	Vote int8   `json:"vote"`
}

type Comment struct {
	Created string `json:"created"`
	Author  Author `json:"author"`
	Body    string `json:"body"` //Comment
	ID      string `json:"id"`
}

type Post struct {
	Score            int64           `json:"score"`
	Views            uint64          `json:"views"`
	Type             string          `json:"type"`
	Title            string          `json:"title"`
	Author           Author          `json:"author"`
	Category         string          `json:"category"`
	Text             string          `json:"text,omitempty"`
	URL              string          `json:"url,omitempty"`
	VotesFromDB      map[string]Vote `json:"-"`
	Votes            []Vote          `json:"votes"`
	Comments         []Comment       `json:"comments"`
	Created          string          `json:"created"`
	UpvotePercentage uint8           `json:"upvotePercentage"`
	UpvoteNum        uint64          `json:"-"`
	ID               string          `json:"id"`
}

type PostRepo interface {
	GetAll() ([]Post, error)
	GetByID(id string) (Post, error)
	Add(item *Post) (uint64, error)
	AddComment(postID, newCom string, sess session.Session) (Post, error)
	DeleteComment(postID, newCom string, sess session.Session) (Post, error)
	Vote(postID, userID string, newVote int8) (Post, error)
	Update(newItem Post) error
	Delete(id string) error
}
