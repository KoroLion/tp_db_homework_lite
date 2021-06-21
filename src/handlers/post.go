package handlers

import (
    "fmt"
    "strconv"
    "net/http"
    "encoding/json"
    "strings"

    "github.com/labstack/echo/v4"
    "github.com/jackc/pgx"

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

    var forumId int
    err = db.QueryRow(`
        UPDATE forums SET posts = posts + $2 WHERE slug = $1
        RETURNING id`,
        forumSlug, len(posts),
    ).Scan(&forumId)
    if err != nil {
        return echo.NewHTTPError(http.StatusNotFound, "Forum was not found!")
    }

    newPosts := make([]models.Post, 0)
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
        if i != len(posts) - 1 {
            queryValues += ", "
        }
        queryParams = append(queryParams, post.Author, post.Message, post.Thread, post.Forum, post.Parent)

        if !utils.IntInList(authorId, userIds) {
            userIds = append(userIds, authorId)
        }

        newPosts = append(newPosts, post)
    }

    tx, err := db.Begin()
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
    }
    defer tx.Rollback()
    if len(queryValues) > 0 {
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

        var queryValues string
        var queryParams []interface{}
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
    }
    tx.Commit()

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
        err = db.QueryRow(`
            SELECT forum FROM threads WHERE id = $1 LIMIT 1`,
            threadId,
        ).Scan(&forumSlug)
    } else {
        err = db.QueryRow(`
            SELECT id, forum FROM threads WHERE slug = $1 LIMIT 1`,
            threadSlug,
        ).Scan(&threadId, &forumSlug)
    }
    if err != nil {
        return echo.NewHTTPError(http.StatusNotFound, "Thread was not found!")
    }

    hasSince := since > 0
    var rows *pgx.Rows
    if desc {
        if sort == "flat" {
            if (hasSince) {
                rows, err = db.Query("post_list_desc_flat_since", threadId, limit, since)
            } else {
                rows, err = db.Query("post_list_desc_flat", threadId, limit)
            }
        } else if sort == "tree" {
            if (hasSince) {
                rows, err = db.Query("post_list_desc_tree_since", threadId, limit, since)
            } else {
                rows, err = db.Query("post_list_desc_tree", threadId, limit)
            }
        } else if sort == "parent_tree" {
            if (hasSince) {
                rows, err = db.Query("post_list_desc_parent_tree_since", threadId, limit, since)
            } else {
                rows, err = db.Query("post_list_desc_parent_tree", threadId, limit)
            }
        }
    } else {
        if sort == "flat" {
            if (hasSince) {
                rows, err = db.Query("post_list_asc_flat_since", threadId, limit, since)
            } else {
                rows, err = db.Query("post_list_asc_flat", threadId, limit)
            }
        } else if sort == "tree" {
            if (hasSince) {
                rows, err = db.Query("post_list_asc_tree_since", threadId, limit, since)
            } else {
                rows, err = db.Query("post_list_asc_tree", threadId, limit)
            }
        } else if sort == "parent_tree" {
            if (hasSince) {
                rows, err = db.Query("post_list_asc_parent_tree_since", threadId, limit, since)
            } else {
                rows, err = db.Query("post_list_asc_parent_tree", threadId, limit)
            }
        }
    }
    if err != nil {
        return echo.NewHTTPError(http.StatusNotFound, err.Error())
    }
    defer rows.Close()

    posts := make([]models.Post, 0)
    for rows.Next() {
        post := models.Post{Forum: forumSlug}
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

    err = db.QueryRow(`
        SELECT parent, author, created, forum, id, message, thread, is_edited
        FROM posts
        WHERE id = $1
        LIMIT 1`,
        post.Id,
    ).Scan(&post.Parent, &post.Author, &post.Created, &post.Forum, &post.Id, &post.Message, &post.Thread, &post.IsEdited)
    if err != nil {
        return echo.NewHTTPError(http.StatusNotFound, err.Error())
    }
    details["post"] = post

    if utils.StringInList("user", related) {
        author := models.User{}
        err = db.QueryRow(`
            SELECT about, email, fullname, nickname
            FROM users
            WHERE nickname = $1
            LIMIT 1`,
            post.Author,
        ).Scan(&author.About, &author.Email, &author.Fullname, &author.Nickname)
        if err != nil {
            return echo.NewHTTPError(http.StatusNotFound, err.Error())
        }
        details["author"] = author
    }

    if utils.StringInList("thread", related) {
        thread := models.Thread{}
        err = db.QueryRow(`
            SELECT author, created, forum, id, message, slug, title, votes
            FROM threads
            WHERE id = $1
            LIMIT 1`,
            post.Thread,
        ).Scan(&thread.Author, &thread.Created, &thread.Forum, &thread.Id, &thread.Message, &thread.Slug, &thread.Title, &thread.Votes)
        if err != nil {
            return echo.NewHTTPError(http.StatusNotFound, err.Error())
        }
        details["thread"] = thread
    }

    if utils.StringInList("forum", related) {
        forum := models.Forum{}
        err = db.QueryRow(`
            SELECT posts, slug, threads, title, user_nickname
            FROM forums
            WHERE slug = $1
            LIMIT 1`,
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
