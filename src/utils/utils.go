package utils

import (
    // _ "github.com/lib/pq"
	"github.com/labstack/echo/v4"
	"github.com/jackc/pgx"
	"tp_db_homework/src/statements"
    "fmt"
    "time"
	"log"
)

type ContextAndDb struct {
	echo.Context
	DB *pgx.ConnPool
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

func PostgresConnect(host string, port int, db_name string, username string, password string) (*pgx.ConnPool, error) {
    log.Println("Connecting to the database!")
    dsn := fmt.Sprintf("user=%s password=%s host=%s port=%d dbname=%s sslmode=disable", username, password, host, port, db_name)

	conn, err := pgx.ParseConnectionString(dsn)
	if err != nil {
		return nil, err
	}
	config := pgx.ConnPoolConfig{
		ConnConfig: conn,
		MaxConnections: 50,
	}
	db, err := pgx.NewConnPool(config)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func PrepareQueries(db *pgx.ConnPool) error {
	err := statements.PostPrepare(db)
	if err != nil {
		return err
	}

	return nil
}

func ClearTables(db *pgx.ConnPool) error {
	_, err := db.Exec(`
		DELETE FROM posts;
		DELETE FROM thread_votes;
		DELETE FROM threads;
		DELETE FROM forum_users;
		DELETE FROM forums;
		DELETE FROM users;
    `)
	return err
}

func ClearDB(db *pgx.ConnPool) error {
	_, err := db.Exec(`
		DROP TABLE IF EXISTS posts;
		DROP TABLE IF EXISTS thread_votes;
		DROP TABLE IF EXISTS threads;
		DROP TABLE IF EXISTS forum_users;
		DROP TABLE IF EXISTS forums;
		DROP TABLE IF EXISTS users;
		DROP TABLE IF EXISTS status;
    `)

    return err
}

func CreateTables(db *pgx.ConnPool) error {
	_, err := db.Exec(`
        CREATE UNLOGGED TABLE IF NOT EXISTS users (
            id SERIAL PRIMARY KEY,
            nickname CITEXT UNIQUE,
            fullname VARCHAR(255),
            about TEXT,
            email CITEXT UNIQUE
        );
		CREATE INDEX IF NOT EXISTS users_nickname ON users USING HASH (nickname);
		CREATE INDEX IF NOT EXISTS users_email ON users USING HASH (email);

        CREATE UNLOGGED TABLE IF NOT EXISTS forums (
            id SERIAL PRIMARY KEY,
            user_nickname CITEXT,
            title VARCHAR(255),
            slug CITEXT UNIQUE,
			threads INT DEFAULT 0,
			posts INT DEFAULT 0,

			FOREIGN KEY (user_nickname) REFERENCES users (nickname)
        );
		CREATE INDEX IF NOT EXISTS forums_slug ON forums USING HASH (slug);

		CREATE UNLOGGED TABLE IF NOT EXISTS forum_users (
			forum_id INT,
			user_id INT,

			UNIQUE(forum_id, user_id),

			FOREIGN KEY (forum_id) REFERENCES forums (id),
			FOREIGN KEY (user_id) REFERENCES users (id)
		);

        CREATE UNLOGGED TABLE IF NOT EXISTS threads (
            id SERIAL PRIMARY KEY,
            forum CITEXT,
            title VARCHAR(255),
            author CITEXT,
            message TEXT,
            created TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            votes INT DEFAULT 0,
            slug CITEXT,

			FOREIGN KEY (forum) REFERENCES forums (slug),
			FOREIGN KEY (author) REFERENCES users (nickname)
        );
		CREATE INDEX IF NOT EXISTS threads_slug ON threads USING HASH (slug);
		CREATE INDEX IF NOT EXISTS threads_forum_created ON threads (forum, created);

        CREATE UNLOGGED TABLE IF NOT EXISTS posts (
            id SERIAL PRIMARY KEY,
            parent INT,
			path INT[],
            author CITEXT,
            message TEXT,
            is_edited BOOLEAN DEFAULT false,
            forum CITEXT,
            thread INT,
            created TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

			FOREIGN KEY (author) REFERENCES users (nickname),
			FOREIGN KEY (forum) REFERENCES forums (slug),
			FOREIGN KEY (thread) REFERENCES threads (id)
        );
		CREATE INDEX IF NOT EXISTS post_path_gin ON posts USING GIN (path);
		CREATE INDEX IF NOT EXISTS post_thread ON posts (thread);
		CREATE INDEX IF NOT EXISTS post_thread_id ON posts (thread, id);
		CREATE INDEX IF NOT EXISTS post_thread_parent_path2 ON posts (thread, parent, (path[2]));

		CREATE UNLOGGED TABLE IF NOT EXISTS thread_votes (
			id SERIAL PRIMARY KEY,
			thread_id INT,
			user_id INT,
			voice INT,

			FOREIGN KEY (thread_id) REFERENCES threads (id),
			FOREIGN KEY (user_id) REFERENCES users (id)
		);
		CREATE INDEX IF NOT EXISTS thread_votes_thread_nickname ON thread_votes (thread_id, user_id);

		CREATE UNLOGGED TABLE IF NOT EXISTS status (
			users INT,
			forums INT,
			threads INT,
			posts INT
		);
		INSERT INTO status (users, forums, threads, posts) VALUES (0, 0, 0, 0);

		CREATE OR REPLACE FUNCTION update_path()
			RETURNS TRIGGER
			AS $update_path$
		DECLARE
		BEGIN
			NEW.path = array_append(COALESCE((SELECT path FROM posts WHERE id = NEW.parent), ARRAY[0]), NEW.id);
		RETURN NEW;
		END;
		$update_path$ LANGUAGE plpgsql;

		DROP TRIGGER IF EXISTS posts_path ON posts;
		CREATE TRIGGER posts_path BEFORE INSERT ON posts
			FOR EACH ROW
			EXECUTE PROCEDURE update_path();
    `)

    return err
}
