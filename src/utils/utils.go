package utils

import (
	"context"
    _ "github.com/lib/pq"
	"github.com/labstack/echo/v4"
	"github.com/jackc/pgx/v4/pgxpool"
    "fmt"
    "time"
)

type ContextAndDb struct {
	echo.Context
	DB *pgxpool.Pool
}

func StringInList(s string, list []string) bool {
    for _, el := range list {
        if s == el {
            return true
        }
    }
    return false
}

func IntInList(i int, list []int) bool {
	for _, el := range list {
        if i == el {
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

func PostgresConnect(host string, port int, db_name string, username string, password string) (*pgxpool.Pool, error) {
    fmt.Println("Connecting to the database!")
    dsn := fmt.Sprintf("user=%s password=%s host=%s port=%d dbname=%s sslmode=disable", username, password, host, port, db_name)

	db, err := pgxpool.Connect(context.Background(), dsn)
	if err != nil {
		return nil, err
	}

	err = db.Ping(context.Background())
	if err != nil {
		return nil, err
	}

	return db, nil
}

func ClearTables(db *pgxpool.Pool) error {
	_, err := db.Exec(context.Background(), `
		DELETE FROM posts;
		DELETE FROM thread_votes;
		DELETE FROM threads;
		DELETE FROM forum_users;
		DELETE FROM forums;
		DELETE FROM users;
    `)
	return err
}

func ClearDB(db *pgxpool.Pool) error {
	_, err := db.Exec(context.Background(), `
		DROP TABLE IF EXISTS posts;
		DROP TABLE IF EXISTS thread_votes;
		DROP TABLE IF EXISTS threads;
		DROP TABLE IF EXISTS forum_users;
		DROP TABLE IF EXISTS forums;
		DROP TABLE IF EXISTS users;
    `)

    return err
}

func CreateTables(db *pgxpool.Pool) error {
	_, err := db.Exec(context.Background(), `
        CREATE UNLOGGED TABLE IF NOT EXISTS users (
            id BIGSERIAL PRIMARY KEY,
            nickname CITEXT UNIQUE,
            fullname VARCHAR(255),
            about TEXT,
            email CITEXT UNIQUE
        );
		CREATE INDEX users_nickname ON users USING HASH (nickname);

        CREATE UNLOGGED TABLE IF NOT EXISTS forums (
            id BIGSERIAL PRIMARY KEY,
            user_nickname CITEXT,
            title VARCHAR(255),
            slug CITEXT UNIQUE,
			threads BIGINT DEFAULT 0,
			posts BIGINT DEFAULT 0,

			FOREIGN KEY (user_nickname) REFERENCES users (nickname)
        );
		CREATE INDEX forums_slug ON forums USING HASH (slug);

		CREATE UNLOGGED TABLE IF NOT EXISTS forum_users (
			forum_id BIGINT,
			user_id BIGINT,

			UNIQUE(forum_id, user_id),

			FOREIGN KEY (forum_id) REFERENCES forums (id),
			FOREIGN KEY (user_id) REFERENCES users (id)
		);

        CREATE UNLOGGED TABLE IF NOT EXISTS threads (
            id BIGSERIAL PRIMARY KEY,
            forum CITEXT,
            title VARCHAR(255),
            author CITEXT,
            message TEXT,
            created TIMESTAMP WITH TIME ZONE,
            votes BIGINT DEFAULT 0,
            slug CITEXT,

			FOREIGN KEY (forum) REFERENCES forums (slug),
			FOREIGN KEY (author) REFERENCES users (nickname)
        );
		CREATE INDEX threads_slug ON threads USING HASH (slug);

        CREATE UNLOGGED TABLE IF NOT EXISTS posts (
            id BIGSERIAL PRIMARY KEY,
            parent BIGINT,
			path BIGINT[],
            author CITEXT,
            message TEXT,
            is_edited BOOLEAN DEFAULT false,
            forum CITEXT,
            thread BIGINT,
            created TIMESTAMP,

			FOREIGN KEY (author) REFERENCES users (nickname),
			FOREIGN KEY (forum) REFERENCES forums (slug),
			FOREIGN KEY (thread) REFERENCES threads (id)
        );

		CREATE UNLOGGED TABLE IF NOT EXISTS thread_votes (
			id BIGSERIAL PRIMARY KEY,
			thread_id BIGINT,
			user_id BIGINT,
			voice BIGINT,

			UNIQUE(thread_id, user_id),

			FOREIGN KEY (thread_id) REFERENCES threads (id),
			FOREIGN KEY (user_id) REFERENCES users (id)
		);
		CREATE INDEX thread_votes_thread_nickname ON thread_votes (thread_id, user_id);

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

    return err
}
