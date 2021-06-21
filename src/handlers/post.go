package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx"
	"github.com/labstack/echo/v4"

	"tp_db_homework/src/models"
	"tp_db_homework/src/utils"
)

func PostCreate(c echo.Context) error {
	db := c.(*utils.ContextAndDb).DB

	threadSlug := c.Param("slug_or_id")
	threadId, err := strconv.Atoi(threadSlug)
	if err != nil {
		threadId = 0
	}

	var forumSlug string
	err = db.QueryRow(`
        SELECT id, forum FROM threads WHERE slug = $1 OR id = $2`,
		threadSlug, threadId,
	).Scan(&threadId, &forumSlug)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Thread was not found!")
	}

	posts := make([]models.Post, 0)
	defer c.Request().Body.Close()
	err = json.NewDecoder(c.Request().Body).Decode(&posts)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	newPosts := make([]models.Post, 0)
	newPostsAmount := len(posts)

	if newPostsAmount > 0 {
		tx, err := db.Begin()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		defer tx.Rollback()

		var forumId int
		err = tx.QueryRow(`
            UPDATE forums SET posts = posts + $2 WHERE slug = $1
            RETURNING id`,
			forumSlug, newPostsAmount,
		).Scan(&forumId)
		if err != nil {
			tx.Rollback()
			return echo.NewHTTPError(http.StatusNotFound, "Forum was not found!")
		}

		var userIds []int
		var queryValues string
		var queryParams []interface{}
		for i, post := range posts {
			post.Thread = threadId
			post.Forum = forumSlug

			if post.Parent != 0 {
				var parentThread int
				err := db.QueryRow(`
                    SELECT thread FROM posts WHERE id = $1`,
					post.Parent,
				).Scan(&parentThread)
				if err != nil || post.Thread != parentThread {
					return echo.NewHTTPError(http.StatusConflict, "Parent was not found or created in another thread!")
				}
			}

			var authorId int
			err := db.QueryRow(`
                SELECT id, nickname FROM users WHERE nickname = $1`,
				post.Author,
			).Scan(&authorId, &post.Author)
			if err != nil {
				return echo.NewHTTPError(http.StatusNotFound, err.Error())
			}

			queryValues += fmt.Sprintf(
				"($%d, $%d, $%d, $%d, $%d)",
				i*5+1, i*5+2, i*5+3, i*5+4, i*5+5,
			)
			if i != len(posts)-1 {
				queryValues += ", "
			}
			queryParams = append(queryParams, post.Author, post.Message, post.Thread, post.Forum, post.Parent)

			if !utils.IntInList(authorId, userIds) {
				userIds = append(userIds, authorId)
			}

			newPosts = append(newPosts, post)
		}

		query := fmt.Sprintf(`
            INSERT INTO posts (author, message, thread, forum, parent)
            VALUES %s
            RETURNING id, created`,
			queryValues,
		)
		rows, err := tx.Query(query, queryParams...)
		if err != nil {
			tx.Rollback()
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		curPostInd := 0
		for rows.Next() {
			rows.Scan(&newPosts[curPostInd].Id, &newPosts[curPostInd].Created)
			curPostInd += 1
		}
		rows.Close()

		queryValues = ""
		queryParams = nil
		last := len(userIds) - 1
		for i, userId := range userIds {
			queryValues += fmt.Sprintf(
				"($%d, $%d)",
				i*2+1, i*2+2,
			)
			if i != last {
				queryValues += ", "
			}
			queryParams = append(queryParams, forumId, userId)
		}

		query = fmt.Sprintf(`
            INSERT INTO forum_users (forum_id, user_id)
            VALUES %s
            ON CONFLICT DO NOTHING`,
			queryValues,
		)
		_, err = tx.Exec(query, queryParams...)
		if err != nil {
			tx.Rollback()
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		_, err = tx.Exec("UPDATE status SET posts = posts + $1", newPostsAmount)
		if err != nil {
			tx.Rollback()
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		tx.Commit()
	}

	return c.JSON(http.StatusCreated, newPosts)
}

func PostList(c echo.Context) error {
	db := c.(*utils.ContextAndDb).DB

	sort := c.QueryParam("sort")
	if len(sort) == 0 {
		sort = "flat"
	}

	limit, err := strconv.Atoi(c.QueryParam("limit"))
	if err != nil {
		limit = 100
	}

	desc, err := strconv.ParseBool(c.QueryParam("desc"))
	if err != nil {
		desc = false
	}

	since, err := strconv.Atoi(c.QueryParam("since"))
	if err != nil || since == 0 {
		since = 0
	}

	var forumSlug string
	threadSlug := c.Param("slug_or_id")
	threadId, err := strconv.Atoi(threadSlug)
	if err == nil {
		err = db.QueryRow("thread_get_forum_by_id", threadId).Scan(&forumSlug)
	} else {
		err = db.QueryRow("thread_get_id_forum_by_slug", threadSlug).Scan(&threadId, &forumSlug)
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}

	hasSince := since > 0
	var rows *pgx.Rows
	orderStr := "asc"
	if desc {
		orderStr = "desc"
	}
	queryStr := "post_list_" + orderStr + "_" + sort
	if hasSince {
		rows, err = db.Query(queryStr+"_since", threadId, limit, since)
	} else {
		rows, err = db.Query(queryStr, threadId, limit)
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}
	defer rows.Close()

	posts := make([]models.Post, 0)
	for rows.Next() {
		post := models.Post{Forum: forumSlug}
		err := rows.Scan(&post.Author, &post.Created, &post.Forum, &post.Id, &post.Message, &post.Thread, &post.Parent)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		posts = append(posts, post)
	}

	return c.JSON(http.StatusOK, posts)
}

func PostDetails(c echo.Context) error {
	db := c.(*utils.ContextAndDb).DB

	details := make(map[string]interface{})
	post := models.Post{}

	related := strings.Split(c.QueryParam("related"), ",")

	var err error
	post.Id, err = strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	err = db.QueryRow("post_get_by_id", post.Id).Scan(&post.Parent, &post.Author, &post.Created, &post.Forum, &post.Id, &post.Message, &post.Thread, &post.IsEdited)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}
	details["post"] = post

	if utils.StringInList("user", related) {
		author := models.User{}
		err = db.QueryRow("user_get_by_nickname", post.Author).Scan(&author.Nickname, &author.Fullname, &author.About, &author.Email)
		if err != nil {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		details["author"] = author
	}

	if utils.StringInList("thread", related) {
		thread := models.Thread{Id: post.Thread}

		err = db.QueryRow("thread_get_by_id", thread.Id).Scan(&thread.Author, &thread.Created, &thread.Forum, &thread.Message, &thread.Slug, &thread.Title, &thread.Votes)
		if err != nil {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		details["thread"] = thread
	}

	if utils.StringInList("forum", related) {
		forum := models.Forum{}
		err = db.QueryRow("forum_get_by_slug", post.Forum).Scan(&forum.Slug, &forum.Title, &forum.User, &forum.Threads, &forum.Posts)
		if err != nil {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		details["forum"] = forum
	}

	return c.JSON(http.StatusOK, details)
}

func PostUpdate(c echo.Context) error {
	db := c.(*utils.ContextAndDb).DB
	post := models.Post{}

	var err error
	post.Id, err = strconv.Atoi(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	postUpd := models.PostUpdate{}
	defer c.Request().Body.Close()
	err = json.NewDecoder(c.Request().Body).Decode(&postUpd)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	err = db.QueryRow(`
        UPDATE posts SET
            message = COALESCE($2, message),
            is_edited = CASE WHEN $2 IS NOT NULL AND message != $2 THEN true ELSE false END
        WHERE id = $1
        RETURNING author, created, forum, id, message, thread, is_edited`,
		post.Id, postUpd.Message,
	).Scan(&post.Author, &post.Created, &post.Forum, &post.Id, &post.Message, &post.Thread, &post.IsEdited)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}

	return c.JSON(http.StatusOK, post)
}
