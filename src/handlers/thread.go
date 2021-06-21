package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx"
	"github.com/labstack/echo/v4"

	"tp_db_homework/src/models"
	"tp_db_homework/src/utils"
)

func ThreadCreate(c echo.Context) error {
	db := c.(*utils.ContextAndDb).DB

	newThread := models.Thread{}
	newThread.Forum = c.Param("slug")

	defer c.Request().Body.Close()
	err := json.NewDecoder(c.Request().Body).Decode(&newThread)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var authorId int
	err = db.QueryRow(`
        SELECT id, nickname FROM users WHERE nickname = $1`,
		newThread.Author,
	).Scan(&authorId, &newThread.Author)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "User not found!")
	}

	if len(newThread.Slug) > 0 {
		var oldThread models.Thread
		err = db.QueryRow(`
            SELECT id, author, created, forum, message, slug, title FROM threads WHERE slug = $1`,
			newThread.Slug,
		).Scan(&oldThread.Id, &oldThread.Author, &oldThread.Created, &oldThread.Forum, &oldThread.Message, &oldThread.Slug, &oldThread.Title)
		if err == nil {
			return echo.NewHTTPError(http.StatusConflict, oldThread)
		}
	}

	var forumId int
	err = db.QueryRow(`
        UPDATE forums SET threads = threads + 1 WHERE slug = $1
        RETURNING id, slug`,
		newThread.Forum,
	).Scan(&forumId, &newThread.Forum)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Forum was not found!")
	}

	err = db.QueryRow(`
        INSERT INTO threads (forum, title, author, message, created, slug) VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id`,
		newThread.Forum, newThread.Title, newThread.Author, newThread.Message, newThread.Created, newThread.Slug,
	).Scan(&newThread.Id)
	if err != nil {
		log.Println(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	db.Exec(`
        INSERT INTO forum_users (forum_id, user_id) VALUES ($1, $2)`,
		forumId, authorId,
	)
	_, err = db.Exec(`UPDATE status SET threads = threads + 1`)

	return c.JSON(http.StatusCreated, newThread)
}

func ThreadList(c echo.Context) error {
	db := c.(*utils.ContextAndDb).DB

	forumSlug := c.Param("slug")
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	desc, _ := strconv.ParseBool(c.QueryParam("desc"))

	err := db.QueryRow("forum_get_slug_by_slug", forumSlug).Scan(&forumSlug)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Forum was not found")
	}

	hasSince := true
	since, err := time.Parse(time.RFC3339, c.QueryParam("since"))
	if err != nil {
		hasSince = false
	}

	orderStr := "asc"
	if desc {
		orderStr = "desc"
	}
	var rows *pgx.Rows
	if hasSince {
		rows, err = db.Query("thread_list_"+orderStr+"_since", forumSlug, limit, since)
	} else {
		rows, err = db.Query("thread_list_"+orderStr, forumSlug, limit)
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	defer rows.Close()

	threads := make([]models.Thread, 0)
	for rows.Next() {
		thr := models.Thread{}
		err := rows.Scan(&thr.Author, &thr.Created, &thr.Forum, &thr.Id, &thr.Message, &thr.Slug, &thr.Title, &thr.Votes)

		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		threads = append(threads, thr)
	}

	return c.JSON(http.StatusOK, threads)
}

func ThreadVote(c echo.Context) error {
	db := c.(*utils.ContextAndDb).DB

	threadSlug := c.Param("slug_or_id")
	threadId, err := strconv.Atoi(threadSlug)
	if err != nil {
		threadId = 0
	}

	thr := models.Thread{}
	err = db.QueryRow(`
        SELECT author, created, forum, id, message, slug, title, votes
        FROM threads
        WHERE slug = $1 OR id = $2`,
		threadSlug, threadId,
	).Scan(&thr.Author, &thr.Created, &thr.Forum, &thr.Id, &thr.Message, &thr.Slug, &thr.Title, &thr.Votes)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}

	var thrVote models.ThreadVote
	defer c.Request().Body.Close()
	err = json.NewDecoder(c.Request().Body).Decode(&thrVote)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if thrVote.Voice > 1 || thrVote.Voice < -1 {
		log.Println("abs(voice) > 1")
		return echo.NewHTTPError(http.StatusBadRequest, "abs(voice) > 1")
	}

	var userId int
	err = db.QueryRow(`
        SELECT id, nickname FROM users WHERE nickname = $1 LIMIT 1`,
		thrVote.Nickname,
	).Scan(&userId, &thrVote.Nickname)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}

	tx, err := db.Begin()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	defer tx.Rollback()

	var prevVoice int
	err = db.QueryRow(`
        SELECT voice FROM thread_votes WHERE thread_id = $1 AND user_id = $2 LIMIT 1`,
		thr.Id, userId,
	).Scan(&prevVoice)
	if err != nil {
		prevVoice = 0
		if thr.Id <= 0 || userId <= 0 {
			log.Printf("No thread_votes for %d and %d", thr.Id, userId)
		}
		prevVoice = 0
		_, err := tx.Exec(`
            INSERT INTO thread_votes (thread_id, user_id, voice) VALUES ($1, $2, $3)`,
			thr.Id, userId, thrVote.Voice,
		)
		if err != nil {
			log.Println(err)
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	} else if prevVoice != thrVote.Voice {
		_, err := tx.Exec(`
            UPDATE thread_votes SET voice = $3 WHERE thread_id = $1 AND user_id = $2`,
			thr.Id, userId, thrVote.Voice,
		)
		if err != nil {
			log.Println(err)
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}

	if prevVoice != thrVote.Voice {
		err = tx.QueryRow(`
            UPDATE threads SET
                votes = votes + $2
            WHERE id = $1
            RETURNING votes`,
			thr.Id, thrVote.Voice-prevVoice,
		).Scan(&thr.Votes)
		if err != nil {
			log.Println(err)
		}
	}
	tx.Commit()

	return c.JSON(http.StatusOK, thr)
}

func ThreadDetails(c echo.Context) error {
	db := c.(*utils.ContextAndDb).DB

	var err error
	thr := models.Thread{}
	thr.Slug = c.Param("slug_or_id")
	thr.Id, err = strconv.Atoi(thr.Slug)
	if err == nil {
		err = db.QueryRow("thread_get_by_id", thr.Id).Scan(&thr.Author, &thr.Created, &thr.Forum, &thr.Message, &thr.Slug, &thr.Title, &thr.Votes)
	} else {
		err = db.QueryRow("thread_get_by_slug", thr.Slug).Scan(&thr.Author, &thr.Created, &thr.Forum, &thr.Id, &thr.Message, &thr.Slug, &thr.Title, &thr.Votes)
	}
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}

	return c.JSON(http.StatusOK, thr)
}

func ThreadUpdate(c echo.Context) error {
	db := c.(*utils.ContextAndDb).DB

	threadSlug := c.Param("slug_or_id")
	threadId, err := strconv.Atoi(threadSlug)
	if err != nil {
		threadId = 0
	}

	thrUpd := models.ThreadUpdate{}
	defer c.Request().Body.Close()
	err = json.NewDecoder(c.Request().Body).Decode(&thrUpd)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	thr := models.Thread{}
	err = db.QueryRow(`
        UPDATE threads SET title = COALESCE($3, title), message = COALESCE($4, message)
        WHERE slug = $1 OR id = $2
        RETURNING author, created, forum, id, message, slug, title, votes`,
		threadSlug, threadId, thrUpd.Title, thrUpd.Message,
	).Scan(&thr.Author, &thr.Created, &thr.Forum, &thr.Id, &thr.Message, &thr.Slug, &thr.Title, &thr.Votes)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}
	return c.JSON(http.StatusOK, thr)
}
