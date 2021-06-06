package handlers

import (
    "context"
    "strconv"
    "net/http"
    "encoding/json"
    "strings"

    "github.com/labstack/echo/v4"
    "github.com/jackc/pgx/v4"

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
    err = db.QueryRow(context.Background(), `
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

    var forumId int
    err = db.QueryRow(context.Background(), `
        UPDATE forums SET posts = posts + $2 WHERE slug = $1
        RETURNING id`,
        forumSlug, len(posts),
    ).Scan(&forumId)
    if err != nil {
        return echo.NewHTTPError(http.StatusNotFound, "Forum was not found!")
    }

    newPosts := make([]models.Post, 0)
    var userIds []int
    for _, post := range posts {
        post.Thread = threadId
        post.Forum = forumSlug

        if post.Parent != 0 {
            var parentThread int
            err := db.QueryRow(context.Background(), `
                SELECT thread FROM posts WHERE id = $1`,
                post.Parent,
            ).Scan(&parentThread)
            if err != nil || post.Thread != parentThread {
                return echo.NewHTTPError(http.StatusConflict, "Parent was not found or created in another thread!")
            }
        }

        var authorId int
        err := db.QueryRow(context.Background(), `
            SELECT id, nickname FROM users WHERE nickname = $1`,
            post.Author,
        ).Scan(&authorId, &post.Author)
        if err != nil {
            return echo.NewHTTPError(http.StatusNotFound, err.Error())
        }
        if !utils.IntInList(authorId, userIds) {
            userIds = append(userIds, authorId)
        }

        err = db.QueryRow(context.Background(), `
            INSERT INTO posts (author, message, thread, forum, parent, created)
            VALUES ($1, $2, $3, $4, $5, $6)
            RETURNING id`,
            post.Author, post.Message, post.Thread, post.Forum, post.Parent, post.Created,
        ).Scan(&post.Id)
        if err != nil {
            return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        newPosts = append(newPosts, post)
    }

    for _, userId := range userIds {
        db.Exec(context.Background(), `
            INSERT INTO forum_users (forum_id, user_id) VALUES ($1, $2)`,
            forumId, userId,
        )
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

    threadSlug := c.Param("slug_or_id")
    threadId, err := strconv.Atoi(threadSlug)
    if err != nil {
        threadId = 0
    }

    var forumSlug string
    err = db.QueryRow(context.Background(), `
        SELECT id, forum FROM threads WHERE slug = $1 OR id = $2`,
        threadSlug, threadId,
    ).Scan(&threadId, &forumSlug)
    if err != nil {
        return echo.NewHTTPError(http.StatusNotFound, "Thread was not found!")
    }

    posts := make([]models.Post, 0)
    var rows pgx.Rows
    if sort == "flat" {
        rows, err = db.Query(context.Background(), `
            SELECT author, created, forum, id, message, thread, parent
            FROM posts
            WHERE
                thread = $1
                AND
                (
                    CASE WHEN $3 THEN id < $4 ELSE id > $4 END
                    OR
                    $4 = 0
                )
            ORDER BY
                CASE WHEN $3 THEN id END DESC,
                id ASC
            LIMIT $2`,
            threadId, limit, desc, since,
        )
    } else if sort == "tree" {
        rows, err = db.Query(context.Background(), `
            SELECT author, created, forum, id, message, thread, parent
            FROM posts
            WHERE
                thread = $1
                AND
                CASE WHEN $3 THEN
                    (path < (SELECT path FROM posts WHERE id = $4)
                    OR
                    $4 = 0)
                ELSE
                    path > COALESCE((SELECT path FROM posts WHERE id = $4), ARRAY[0])
                END
            ORDER BY
                CASE WHEN $3 THEN path END DESC,
                path ASC
            LIMIT $2`,
            threadId, limit, desc, since,
        )
    } else if sort == "parent_tree" {
        rows, err = db.Query(context.Background(), `
            SELECT author, created, forum, id, message, thread, parent
            FROM posts
            WHERE path[2] IN (
                SELECT id FROM posts
                WHERE
                    thread = $1 AND parent = 0
                    AND
                    CASE WHEN $3 THEN
                        (path[2] < (SELECT path[2] FROM posts WHERE id = $4)
                        OR
                        $4 = 0)
                    ELSE
                        path[2] > COALESCE((SELECT path[2] FROM posts WHERE id = $4), 0)
                    END
                ORDER BY
                    CASE WHEN $3 THEN id END DESC,
                    id ASC
                LIMIT $2
            )
            ORDER BY
                CASE WHEN $3 THEN path[2] END DESC,
                path`,
            threadId, limit, desc, since,
        )
    }
    if err != nil {
        return echo.NewHTTPError(http.StatusNotFound, err.Error())
    }
    defer rows.Close()
    for rows.Next() {
        post := models.Post{}
        post.Forum = forumSlug
        err := rows.Scan(&post.Author, &post.Created, &post.Forum, &post.Id, &post.Message, &post.Thread, &post.Parent);
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

    err = db.QueryRow(context.Background(), `
        SELECT author, created, forum, id, message, thread, is_edited FROM posts WHERE id = $1`,
        post.Id,
    ).Scan(&post.Author, &post.Created, &post.Forum, &post.Id, &post.Message, &post.Thread, &post.IsEdited)
    if err != nil {
        return echo.NewHTTPError(http.StatusNotFound, err.Error())
    }
    details["post"] = post

    if utils.StringInList("user", related) {
        author := models.User{}
        err = db.QueryRow(context.Background(), `
            SELECT about, email, fullname, nickname FROM users WHERE nickname = $1`,
            post.Author,
        ).Scan(&author.About, &author.Email, &author.Fullname, &author.Nickname)
        if err != nil {
            return echo.NewHTTPError(http.StatusNotFound, err.Error())
        }
        details["author"] = author
    }

    if utils.StringInList("thread", related) {
        thread := models.Thread{}
        err = db.QueryRow(context.Background(), `
            SELECT author, created, forum, id, message, slug, title FROM threads WHERE id = $1`,
            post.Thread,
        ).Scan(&thread.Author, &thread.Created, &thread.Forum, &thread.Id, &thread.Message, &thread.Slug, &thread.Title)
        if err != nil {
            return echo.NewHTTPError(http.StatusNotFound, err.Error())
        }
        details["thread"] = thread
    }

    if utils.StringInList("forum", related) {
        forum := models.Forum{}
        err = db.QueryRow(context.Background(), `
            SELECT posts, slug, threads, title, user_nickname FROM forums WHERE slug = $1`,
            post.Forum,
        ).Scan(&forum.Posts, &forum.Slug, &forum.Threads, &forum.Title, &forum.User)
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

    err = db.QueryRow(context.Background(), `
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
