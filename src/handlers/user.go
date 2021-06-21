package handlers

import (
    "log"
    "net/http"
    "encoding/json"

    "github.com/labstack/echo/v4"

    "tp_db_homework/src/models"
    "tp_db_homework/src/utils"
)

func UserCreate(c echo.Context) error {
    db := c.(*utils.ContextAndDb).DB

    nickname := c.Param("nickname")

    newUser := models.User{}
    defer c.Request().Body.Close()
    newUser.Nickname = nickname

    err := json.NewDecoder(c.Request().Body).Decode(&newUser)
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, err.Error())
    }

    oldUsers := make([]models.User, 0)
    rows1, err := db.Query(`
        SELECT nickname, fullname, about, email
        FROM users
        WHERE nickname = $1 OR email = $2`,
        newUser.Nickname, newUser.Email,
    )
    if err != nil {
        log.Println(err)
        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
    }
    defer rows1.Close()

    for rows1.Next() {
        oldUser := models.User{}
        err := rows1.Scan(&oldUser.Nickname, &oldUser.Fullname, &oldUser.About, &oldUser.Email)
        if err != nil {
            log.Println(err)
            return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }
        oldUsers = append(oldUsers, oldUser)
    }

    if len(oldUsers) > 0 {
        return c.JSON(409, oldUsers)
    }

    _, err = db.Exec(`
        INSERT INTO users (nickname, fullname, about, email) VALUES ($1, $2, $3, $4)`,
        newUser.Nickname, newUser.Fullname, newUser.About, newUser.Email,
    )
    if err != nil {
        log.Println(err)
        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
    }
    db.Exec(`UPDATE status SET users = users + 1`)

    return c.JSON(201, newUser)
}

func UserDetails(c echo.Context) error {
    db := c.(*utils.ContextAndDb).DB

    nickname := c.Param("nickname")

    user := models.User{}
    err := db.QueryRow("user_get_by_nickname", nickname,
        ).Scan(&user.Nickname, &user.Fullname, &user.About, &user.Email)
    if err != nil {
        return echo.NewHTTPError(http.StatusNotFound, err.Error())
    }

    return c.JSON(200, user)
}

func UserUpdate(c echo.Context) error {
    db := c.(*utils.ContextAndDb).DB

    nickname := c.Param("nickname")

    var count int
    err := db.QueryRow(`
        SELECT COUNT(*) FROM users WHERE nickname = $1`,
        nickname,
    ).Scan(&count)
    if err != nil {
        log.Println(err)
        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
    }
    if count == 0 {
        return echo.NewHTTPError(http.StatusNotFound, "User not found")
    }

    updatedUser := models.UserUpdate{}
    defer c.Request().Body.Close()
    updatedUser.Nickname = &nickname
    err = json.NewDecoder(c.Request().Body).Decode(&updatedUser)
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, err.Error())
    }

    err = db.QueryRow(`
        SELECT COUNT(*) FROM users WHERE email = $1`,
        updatedUser.Email,
    ).Scan(&count)
    if err != nil {
        log.Println(err)
        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
    }
    if count > 0 {
        return echo.NewHTTPError(http.StatusConflict, "Email already exists")
    }

    user := models.User{}
    err = db.QueryRow(`
        UPDATE users SET fullname = COALESCE($2, fullname), about = COALESCE($3, about), email = COALESCE($4, email)
        WHERE nickname = $1
        RETURNING nickname, fullname, about, email`,
        updatedUser.Nickname, updatedUser.Fullname, updatedUser.About, updatedUser.Email,
    ).Scan(&user.Nickname, &user.Fullname, &user.About, &user.Email)
    if err != nil {
        log.Println(err)
        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
    }

    return c.JSON(200, user)
}
