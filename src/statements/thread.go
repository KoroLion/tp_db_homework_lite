package statements

import (
	"github.com/jackc/pgx"
)

func ThreadPrepare(db *pgx.ConnPool) error {
	_, err := db.Prepare("thread_get_forum_by_id", `SELECT forum FROM threads WHERE id = $1 LIMIT 1`)
	if err != nil {
		return err
	}

	_, err = db.Prepare("thread_get_id_forum_by_slug", `SELECT id, forum FROM threads WHERE slug = $1 LIMIT 1`)
	if err != nil {
		return err
	}

	_, err = db.Prepare("thread_get_by_id", `
        SELECT author, created, forum, message, slug, title, votes
        FROM threads
        WHERE id = $1
        LIMIT 1`,
	)
	if err != nil {
		return err
	}

	_, err = db.Prepare("thread_get_by_slug", `
        SELECT author, created, forum, id, message, slug, title, votes
        FROM threads
        WHERE slug = $1
        LIMIT 1`,
	)
	if err != nil {
		return err
	}

	_, err = db.Prepare("thread_list_desc_since", `
        SELECT author, created, forum, id, message, slug, title, votes FROM threads
        WHERE forum = $1 AND created <= $3
        ORDER BY created DESC
        LIMIT $2`,
	)
	if err != nil {
		return err
	}

	_, err = db.Prepare("thread_list_desc", `
        SELECT author, created, forum, id, message, slug, title, votes FROM threads
        WHERE forum = $1
        ORDER BY created DESC
        LIMIT $2`,
	)
	if err != nil {
		return err
	}

	_, err = db.Prepare("thread_list_asc_since", `
        SELECT author, created, forum, id, message, slug, title, votes FROM threads
        WHERE forum = $1 AND created >= $3
        ORDER BY created ASC
        LIMIT $2`,
	)
	if err != nil {
		return err
	}

	_, err = db.Prepare("thread_list_asc", `
        SELECT author, created, forum, id, message, slug, title, votes FROM threads
        WHERE forum = $1
        ORDER BY created ASC
        LIMIT $2`,
	)
	if err != nil {
		return err
	}

	return nil
}
