package handlers

import (
    "log"
    "net/http"
    "encoding/json"
    "strconv"

    "github.com/labstack/echo/v4"

    "tp_db_homework/src/models"
    "tp_db_homework/src/utils"
)

func ForumCreate(c echo.Context) error {
    db := c.(*utils.ContextAndDb).DB

    newForum := models.Forum{}
    defer c.Request().Body.Close()

    err := json.NewDecoder(c.Request().Body).Decode(&newForum)
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, err.Error())
    }

    err = db.QueryRow(
        "SELECT nickname FROM users WHERE LOWER(nickname) = LOWER($1)",
        newForum.User,
    ).Scan(&newForum.User)
    if err != nil {
        log.Println(err)
        return echo.NewHTTPError(http.StatusNotFound, "User not found!")
    }

    err = db.QueryRow("SELECT title, user_nickname, slug FROM forums WHERE LOWER(slug) = LOWER($1)", newForum.Slug).Scan(&newForum.Title, &newForum.User, &newForum.Slug)
    if err == nil {
        return c.JSON(409, newForum)
    }

    rows, err := db.Query("INSERT INTO forums (title, user_nickname, slug) VALUES ($1, $2, $3)", newForum.Title, newForum.User, newForum.Slug)
    if err != nil {
        log.Println(err)
        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
    }
    defer rows.Close()

    return c.JSON(http.StatusCreated, newForum)
}

func ForumDetails(c echo.Context) error {
    db := c.(*utils.ContextAndDb).DB

    forum := models.Forum{}
    forum.Slug = c.Param("slug")

    err := db.QueryRow(`
        SELECT slug, title, user_nickname, threads, posts FROM forums WHERE LOWER(slug) = LOWER($1)`,
        forum.Slug,
    ).Scan(&forum.Slug, &forum.Title, &forum.User, &forum.Threads, &forum.Posts)
    if err != nil {
        log.Println(err)
        return echo.NewHTTPError(http.StatusNotFound, "Forum not found")
    }

    err = db.QueryRow(`
        SELECT COUNT(*) FROM posts WHERE LOWER(forum) = LOWER($1)`,
        forum.Slug,
    ).Scan(&forum.Posts)
    if err != nil {
        log.Println(err)
        return echo.NewHTTPError(http.StatusNotFound, "Forum not found")
    }

    return c.JSON(http.StatusOK, forum)
}

func ForumUsers(c echo.Context) error {
    db := c.(*utils.ContextAndDb).DB
    forumSlug := c.Param("slug")

    limit, err := strconv.Atoi(c.QueryParam("limit"))
    if err != nil {
        limit = 100
    }

    desc, err := strconv.ParseBool(c.QueryParam("desc"))
    if err != nil {
        desc = false
    }

    since := c.QueryParam("since")

    var forumId int
    err = db.QueryRow(`
        SELECT id FROM forums WHERE LOWER(slug) = LOWER($1)`,
        forumSlug,
    ).Scan(&forumId)
    if err != nil {
        return echo.NewHTTPError(http.StatusNotFound, err.Error())
    }

    rows, err := db.Query(`
        SELECT about, email, fullname, nickname
        FROM forum_users fu
            INNER JOIN users u ON u.id = fu.user_id
        WHERE
            forum_id = $1
            AND
            (
                CASE WHEN $3 THEN nickname < $4 ELSE nickname > $4 END
                OR
                LENGTH($4) = 0
            )
        ORDER BY
            CASE WHEN $3 THEN nickname END DESC,
            nickname ASC
        LIMIT $2`,
        forumId, limit, desc, since,
    )
    if err != nil {
        log.Println(err)
        return echo.NewHTTPError(http.StatusNotFound, err.Error())
    }
    defer rows.Close()

    users := make([]models.User, 0)
    for rows.Next() {
        user := models.User{}
        rows.Scan(&user.About, &user.Email, &user.Fullname, &user.Nickname)
        users = append(users, user)
    }
    return c.JSON(http.StatusOK, users)
}