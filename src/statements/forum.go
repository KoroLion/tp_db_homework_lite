package statements

import (
	"github.com/jackc/pgx"
)

func ForumPrepare(db *pgx.ConnPool) error {
	_, err := db.Prepare("forum_get_id_by_slug", `SELECT id FROM forums WHERE slug = $1`)
	if err != nil {
		return err
	}

	_, err = db.Prepare("forum_get_slug_by_slug", `SELECT slug FROM forums WHERE slug = $1`)
	if err != nil {
		return err
	}

	_, err = db.Prepare("forum_get_by_slug", `
        SELECT slug, title, user_nickname, threads, posts
        FROM forums
        WHERE slug = $1
        LIMIT 1`,
	)
	if err != nil {
		return err
	}

	_, err = db.Prepare("forum_users_desc_since", `
        SELECT about, email, fullname, nickname
        FROM forum_users fu
            INNER JOIN users u ON u.id = fu.user_id
        WHERE forum_id = $1 AND nickname < $3
        ORDER BY nickname DESC
        LIMIT $2`,
	)
	if err != nil {
		return err
	}

	_, err = db.Prepare("forum_users_desc", `
        SELECT about, email, fullname, nickname
        FROM forum_users fu
            INNER JOIN users u ON u.id = fu.user_id
        WHERE forum_id = $1
        ORDER BY nickname DESC
        LIMIT $2`,
	)
	if err != nil {
		return err
	}

	_, err = db.Prepare("forum_users_asc_since", `
        SELECT about, email, fullname, nickname
        FROM forum_users fu
            INNER JOIN users u ON u.id = fu.user_id
        WHERE forum_id = $1 AND nickname > $3
        ORDER BY nickname ASC
        LIMIT $2`,
	)
	if err != nil {
		return err
	}

	_, err = db.Prepare("forum_users_asc", `
        SELECT about, email, fullname, nickname
        FROM forum_users fu
            INNER JOIN users u ON u.id = fu.user_id
        WHERE forum_id = $1
        ORDER BY nickname ASC
        LIMIT $2`,
	)
	if err != nil {
		return err
	}

	return nil
}
