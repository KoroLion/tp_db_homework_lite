package handlers

import (
    "context"
    "log"
    "net/http"

    "github.com/labstack/echo/v4"

    "tp_db_homework/src/models"
    "tp_db_homework/src/utils"
)

func ServiceClear(c echo.Context) error {
    db := c.(*utils.ContextAndDb).DB

    err := utils.ClearTables(db)
    if err != nil {
        log.Println(err)
        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
    }

    return c.String(200, "")
}

func ServiceStatus(c echo.Context) error {
    db := c.(*utils.ContextAndDb).DB

    serviceStatus := models.ServiceStatus{}
    err1 := db.QueryRow(context.Background(), "SELECT COUNT(*) FROM users").Scan(&serviceStatus.UserCount)
    err2 := db.QueryRow(context.Background(), "SELECT COUNT(*) FROM forums").Scan(&serviceStatus.ForumCount)
    err3 := db.QueryRow(context.Background(), "SELECT COUNT(*) FROM threads").Scan(&serviceStatus.ThreadCount)
    err4 := db.QueryRow(context.Background(), "SELECT COUNT(*) FROM posts").Scan(&serviceStatus.PostCount)
    if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
        log.Println(err1)
        return echo.NewHTTPError(http.StatusInternalServerError, err1.Error())
    }
    return c.JSON(200, serviceStatus)
}
