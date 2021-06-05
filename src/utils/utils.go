package utils

import (
	"database/sql"
    _ "github.com/lib/pq"
	"github.com/labstack/echo/v4"
    "fmt"
    "time"
)

type ContextAndDb struct {
	echo.Context
	DB *sql.DB
}

func StringInList(s string, list []string) bool {
    for _, el := range list {
        if s == el {
            return true
        }
    }
    return false
}

func GetSpecialDate(max bool) time.Time {
    if max {
        t, _ := time.Parse(time.RFC3339, "9999-12-31T00:00:00.000+00:00")
        return t
    } else {
        return time.Time{}
    }
}

func PostgresConnect(host string, port int, db_name string, username string, password string) (*sql.DB, error) {
    fmt.Println("Connecting to the database!")
    dsn := fmt.Sprintf("user=%s password=%s host=%s port=%d dbname=%s sslmode=disable", username, password, host, port, db_name)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil
}

func ClearTables(db *sql.DB) error {
	rows, err := db.Query(`
		DELETE FROM users;
		DELETE FROM forums;
		DELETE FROM threads;
		DELETE FROM posts;
    `)
	rows.Close()
	return err
}

func ClearDB(db *sql.DB) error {
	rows, err := db.Query(`
		DROP TABLE IF EXISTS users;
		DROP TABLE IF EXISTS forums;
		DROP TABLE IF EXISTS forum_users;
		DROP TABLE IF EXISTS threads;
		DROP TABLE IF EXISTS posts;
		DROP TABLE IF EXISTS thread_votes;

		DROP TRIGGER IF EXISTS posts_path ON posts;
    `)
	rows.Close()
    return err
}

func CreateTables(db *sql.DB) error {
	rows, err := db.Query(`
        CREATE TABLE IF NOT EXISTS users (
            id SERIAL PRIMARY KEY,
            nickname VARCHAR(255) UNIQUE,
            fullname VARCHAR(255),
            about TEXT,
            email VARCHAR(255) UNIQUE
        );

        CREATE TABLE IF NOT EXISTS forums (
            id SERIAL PRIMARY KEY,
            user_nickname VARCHAR(255),
            title VARCHAR(255),
            slug VARCHAR(255) UNIQUE,
			threads INT DEFAULT 0,
			posts INT DEFAULT 0
        );

		CREATE TABLE IF NOT EXISTS forum_users (
			forum_id INT,
			user_id INT,
			UNIQUE(forum_id, user_id)
		);

        CREATE TABLE IF NOT EXISTS threads (
            id SERIAL PRIMARY KEY,
            forum VARCHAR(255),
            title VARCHAR(255),
            author VARCHAR(255),
            message TEXT,
            created TIMESTAMP WITH TIME ZONE,
            votes INT DEFAULT 0,
            slug VARCHAR(255)
        );

        CREATE TABLE IF NOT EXISTS posts (
            id SERIAL PRIMARY KEY,
            parent INT,
			path INT[],
            author VARCHAR(255),
            message TEXT,
            is_edited BOOLEAN DEFAULT false,
            forum VARCHAR(255),
            thread INT,
            created TIMESTAMP
        );

		CREATE TABLE IF NOT EXISTS thread_votes (
			id SERIAL PRIMARY KEY,
			thread INT,
			nickname VARCHAR(255),
			voice INT
		);

		CREATE OR REPLACE FUNCTION update_path()
			RETURNS TRIGGER
			AS $update_path$
		DECLARE
		BEGIN
			NEW.path = array_append(COALESCE((SELECT path FROM posts WHERE id = NEW.parent), ARRAY[0]), NEW.id);
		RETURN NEW;
		END;
		$update_path$ LANGUAGE plpgsql;

		CREATE TRIGGER posts_path BEFORE INSERT ON posts
			FOR EACH ROW
			EXECUTE PROCEDURE update_path();
    `)
	rows.Close()
    return err
}
