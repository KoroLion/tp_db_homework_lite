package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/jackc/pgx"
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

	err = db.QueryRow(`
        SELECT nickname
        FROM users
        WHERE nickname = $1`,
		newForum.User,
	).Scan(&newForum.User)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "User not found!")
	}

	err = db.QueryRow(`
        SELECT title, user_nickname, slug
        FROM forums
        WHERE slug = $1`,
		newForum.Slug,
	).Scan(&newForum.Title, &newForum.User, &newForum.Slug)
	if err == nil {
		return c.JSON(409, newForum)
	}

	_, err = db.Exec(`
        INSERT INTO forums (title, user_nickname, slug) VALUES ($1, $2, $3)`,
		newForum.Title, newForum.User, newForum.Slug,
	)
	if err != nil {
		log.Println(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	db.Exec(`UPDATE status SET forums = forums + 1`)

	return c.JSON(http.StatusCreated, newForum)
}

func ForumDetails(c echo.Context) error {
	db := c.(*utils.ContextAndDb).DB

	forum := models.Forum{}
	forum.Slug = c.Param("slug")

	err := db.QueryRow("forum_get_by_slug", forum.Slug).Scan(&forum.Slug, &forum.Title, &forum.User, &forum.Threads, &forum.Posts)
	if err != nil {
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
	err = db.QueryRow("forum_get_id_by_slug", forumSlug).Scan(&forumId)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}

	hasSince := len(since) > 0
	orderStr := "asc"
	if desc {
		orderStr = "desc"
	}

	var rows *pgx.Rows
	if hasSince {
		rows, err = db.Query("forum_users_"+orderStr+"_since", forumId, limit, since)
	} else {
		rows, err = db.Query("forum_users_"+orderStr, forumId, limit)
	}
	if err != nil {
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
