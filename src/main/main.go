package main

import (
	"log"

	"github.com/labstack/echo/v4"
    "github.com/labstack/echo-contrib/prometheus"

	"tp_db_homework/src/handlers"
	"tp_db_homework/src/utils"
)

func main() {
	db, err := utils.PostgresConnect("localhost", 5432, "tp_db_homework", "korolion", "qwerty123")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = utils.CreateTables(db)
	if err != nil {
		log.Fatal(err)
	}
	err = utils.PrepareQueries(db)
	if err != nil {
		log.Fatal(err)
	}

	e := echo.New()
	e.Use(func(h echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cc := &utils.ContextAndDb{c, db}
			return h(cc)
		}
	})
    
    p := prometheus.NewPrometheus("echo", nil)
    p.Use(e)

	e.POST("/api/service/clear", handlers.ServiceClear)
	e.GET("/api/service/status", handlers.ServiceStatus)

	e.POST("/api/user/:nickname/create", handlers.UserCreate)
	e.GET("/api/user/:nickname/profile", handlers.UserDetails)
	e.POST("/api/user/:nickname/profile", handlers.UserUpdate)

	e.POST("/api/forum/create", handlers.ForumCreate)
	e.GET("/api/forum/:slug/details", handlers.ForumDetails)
	e.GET("/api/forum/:slug/users", handlers.ForumUsers)

	e.POST("/api/forum/:slug/create", handlers.ThreadCreate)
	e.GET("/api/forum/:slug/threads", handlers.ThreadList)
	e.POST("/api/thread/:slug_or_id/vote", handlers.ThreadVote)
	e.GET("/api/thread/:slug_or_id/details", handlers.ThreadDetails)
	e.POST("/api/thread/:slug_or_id/details", handlers.ThreadUpdate)

	e.POST("/api/thread/:slug_or_id/create", handlers.PostCreate)
	e.GET("/api/thread/:slug_or_id/posts", handlers.PostList)
	e.GET("/api/post/:id/details", handlers.PostDetails)
	e.POST("/api/post/:id/details", handlers.PostUpdate)

	e.Start(":5000")

	/*quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	go func() {
		err := e.Start(":5000")
		if err != nil {
			log.Println("Server was shut down with no errors!")
		} else {
			log.Fatal("Error occured while trying to shut down server: " + err.Error())
		}
	}()
	<-quit

	log.Println("Interrupt signal received. Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		log.Fatal("Server shut down timeout with an error: " + err.Error())
	}*/
}
