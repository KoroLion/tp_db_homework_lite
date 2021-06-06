package handlers

import (
    "strconv"
    "net/http"
    "encoding/json"
    "time"
    "log"

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
        SELECT id, nickname FROM users WHERE LOWER(nickname) = LOWER($1)`,
        newThread.Author,
    ).Scan(&authorId, &newThread.Author)
    if err != nil {
        log.Println(err)
        return echo.NewHTTPError(http.StatusNotFound, "User not found!")
    }

    if len(newThread.Slug) > 0 {
        var oldThread models.Thread
        err = db.QueryRow(`
            SELECT id, author, created, forum, message, slug, title FROM threads WHERE LOWER(slug) = LOWER($1)`,
            newThread.Slug,
        ).Scan(&oldThread.Id, &oldThread.Author, &oldThread.Created, &oldThread.Forum, &oldThread.Message, &oldThread.Slug, &oldThread.Title)
        if err == nil {
            return echo.NewHTTPError(http.StatusConflict, oldThread)
        }
    }

    var forumId int
    err = db.QueryRow(`
        UPDATE forums SET threads = threads + 1 WHERE LOWER(slug) = LOWER($1)
        RETURNING id, slug`,
        newThread.Forum,
    ).Scan(&forumId, &newThread.Forum)
    if err != nil {
        log.Println(err)
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

    return c.JSON(http.StatusCreated, newThread)
}

func ThreadList(c echo.Context) error {
    db := c.(*utils.ContextAndDb).DB

    forumSlug := c.Param("slug")
    limit, _ := strconv.Atoi(c.QueryParam("limit"))
    desc, _ := strconv.ParseBool(c.QueryParam("desc"))

    var forumCount int64
    err := db.QueryRow(`
        SELECT COUNT(*) FROM forums WHERE LOWER(slug) = LOWER($1)`,
        forumSlug,
    ).Scan(&forumCount)
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
    }
    if forumCount == 0 {
        return echo.NewHTTPError(http.StatusNotFound, "Forum was not found")
    }

    since, err := time.Parse(time.RFC3339, c.QueryParam("since"))
    if err != nil {
        since = utils.GetSpecialDate(desc)
    }
    rows, err := db.Query(`
        SELECT author, created, forum, id, message, slug, title FROM threads
        WHERE LOWER(forum) = LOWER($1) AND CASE WHEN $3 THEN created <= $2 ELSE created >= $2 END
        ORDER BY
            CASE WHEN $3 THEN created END DESC,
            created ASC
        LIMIT $4`,
        forumSlug, since, desc, limit,
    )

    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
    }
    defer rows.Close()

    threads := make([]models.Thread, 0)
    for rows.Next() {
        thr := models.Thread{}
        err := rows.Scan(&thr.Author, &thr.Created, &thr.Forum, &thr.Id, &thr.Message, &thr.Slug, &thr.Title)
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
        err = db.QueryRow(`
            SELECT id, slug FROM threads WHERE LOWER(slug) = LOWER($1) OR id = $2`,
            threadSlug, threadId,
        ).Scan(&threadId, &threadSlug)
        if err != nil {
            return echo.NewHTTPError(http.StatusNotFound, err.Error())
        }
    }

    var thrVote models.ThreadVote
    defer c.Request().Body.Close()
    err = json.NewDecoder(c.Request().Body).Decode(&thrVote)
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, err.Error())
    }

    err = db.QueryRow(`
        SELECT nickname FROM users WHERE LOWER(nickname) = LOWER($1)`,
        thrVote.Nickname,
    ).Scan(&thrVote.Nickname)
    if err != nil {
        return echo.NewHTTPError(http.StatusNotFound, err.Error())
    }

    var prevVoice int
    err = db.QueryRow(`
        SELECT voice FROM thread_votes WHERE thread = $1 AND nickname = $2`,
        threadId, thrVote.Nickname,
    ).Scan(&prevVoice)
    if err != nil {
        _, err := db.Exec(`
            INSERT INTO thread_votes (thread, nickname, voice) VALUES ($1, $2, $3)`,
            threadId, thrVote.Nickname, thrVote.Voice,
        )
        if err != nil {
            return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }
    } else {
        _, err := db.Exec(`
            UPDATE thread_votes SET voice = $3 WHERE thread = $1 AND nickname = $2`,
            threadId, thrVote.Nickname, thrVote.Voice,
        )
        if err != nil {
            return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }
    }

    var thr models.Thread
    err = db.QueryRow(`
        UPDATE threads SET votes = votes - $2 + $3
        WHERE id = $1
        RETURNING author, created, forum, id, message, slug, title, votes`,
        threadId, prevVoice, thrVote.Voice,
    ).Scan(&thr.Author, &thr.Created, &thr.Forum, &thr.Id, &thr.Message, &thr.Slug, &thr.Title, &thr.Votes)
    if err != nil {
        return echo.NewHTTPError(http.StatusNotFound, "Thread was not found!")
    }

    return c.JSON(http.StatusOK, thr)
}

func ThreadDetails(c echo.Context) error {
    db := c.(*utils.ContextAndDb).DB

    threadSlug := c.Param("slug_or_id")
    threadId, err := strconv.Atoi(threadSlug)
    if err != nil {
        threadId = 0
    }
    thr := models.Thread{}
    err = db.QueryRow(`
        SELECT author, created, forum, id, message, slug, title, votes FROM threads WHERE LOWER(slug) = LOWER($1) OR id = $2`,
        threadSlug, threadId,
    ).Scan(&thr.Author, &thr.Created, &thr.Forum, &thr.Id, &thr.Message, &thr.Slug, &thr.Title, &thr.Votes)
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
        WHERE LOWER(slug) = LOWER($1) OR id = $2
        RETURNING author, created, forum, id, message, slug, title, votes`,
        threadSlug, threadId, thrUpd.Title, thrUpd.Message,
    ).Scan(&thr.Author, &thr.Created, &thr.Forum, &thr.Id, &thr.Message, &thr.Slug, &thr.Title, &thr.Votes)
    if err != nil {
        return echo.NewHTTPError(http.StatusNotFound, err.Error())
    }
    return c.JSON(http.StatusOK, thr)
}
