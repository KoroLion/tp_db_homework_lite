package models

import (
	"time"
)

type User struct {
	Nickname string `json:"nickname"`
	Email    string `json:"email"`
	Fullname string `json:"fullname"`
	About    string `json:"about"`
}

type UserUpdate struct {
	Nickname *string `json:"nickname"`
	Email    *string `json:"email"`
	Fullname *string `json:"fullname"`
	About    *string `json:"about"`
}

type Forum struct {
	Title   string `json:"title"`
	User    string `json:"user"`
	Slug    string `json:"slug"`
	Posts   int    `json:"posts"`
	Threads int    `json:"threads"`
}

type ServiceStatus struct {
	UserCount   int `json:"user"`
	ForumCount  int `json:"forum"`
	ThreadCount int `json:"thread"`
	PostCount   int `json:"post"`
}

type Thread struct {
	Id      int       `json:"id"`
	Forum   string    `json:"forum"`
	Title   string    `json:"title"`
	Author  string    `json:"author"`
	Message string    `json:"message"`
	Created time.Time `json:"created"`
	Votes   int       `json:"votes"`
	Slug    string    `json:"slug"`
}

type ThreadUpdate struct {
	Title   *string `json:"title"`
	Message *string `json:"message"`
}

type Post struct {
	Id       int       `json:"id"`
	Parent   int       `json:"parent"`
	Author   string    `json:"author"`
	Message  string    `json:"message"`
	IsEdited bool      `json:"isEdited"`
	Forum    string    `json:"forum"`
	Thread   int       `json:"thread"`
	Created  time.Time `json:"created"`
}

type PostUpdate struct {
	Message *string `json:"message"`
}

type PostDetails struct {
	Post   Post   `json:"post"`
	Thread Thread `json:"thread"`
	Forum  Forum  `json:"forum"`
	User   User   `json:"user"`
}

type ThreadVote struct {
	Id       int    `json:"id"`
	Thread   int    `json:"thread"`
	Nickname string `json:"nickname"`
	Voice    int    `json:"voice"`
}
