package statements

import (
    "github.com/jackc/pgx"
    "log"
)

func PostPrepare(db *pgx.ConnPool) {
    _, err := db.Prepare("post_list_desc_flat", `
		SELECT author, created, forum, id, message, thread, parent
		FROM posts
		WHERE thread = $1
		ORDER BY id DESC
		LIMIT $2`)
	if err != nil { log.Fatal(err) }

	_, err = db.Prepare("post_list_desc_flat_since", `
		SELECT author, created, forum, id, message, thread, parent
		FROM posts
		WHERE thread = $1 AND id < $3
		ORDER BY id DESC
		LIMIT $2`)
	if err != nil { log.Fatal(err) }

	_, err = db.Prepare("post_list_desc_tree_since", `
		SELECT author, created, forum, id, message, thread, parent
		FROM posts
		WHERE thread = $1 AND path < (SELECT path FROM posts WHERE id = $3)
		ORDER BY path DESC
		LIMIT $2`)
	if err != nil { log.Fatal(err) }

	_, err = db.Prepare("post_list_desc_tree", `
		SELECT author, created, forum, id, message, thread, parent
		FROM posts
		WHERE thread = $1
		ORDER BY path DESC
		LIMIT $2`)
	if err != nil { log.Fatal(err) }

	_, err = db.Prepare("post_list_desc_parent_tree_since", `
		SELECT author, created, forum, id, message, thread, parent
		FROM posts
		WHERE path[2] IN (
			SELECT id FROM posts
			WHERE thread = $1 AND parent = 0 AND path[2] < (SELECT path[2] FROM posts WHERE id = $3)
			ORDER BY id DESC
			LIMIT $2
		)
		ORDER BY path[2] DESC, path ASC`)
	if err != nil { log.Fatal(err) }

	_, err = db.Prepare("post_list_desc_parent_tree", `
		SELECT author, created, forum, id, message, thread, parent
		FROM posts
		WHERE path[2] IN (
			SELECT id FROM posts
			WHERE thread = $1 AND parent = 0
			ORDER BY id DESC
			LIMIT $2
		)
		ORDER BY path[2] DESC, path ASC`)
	if err != nil { log.Fatal(err) }

	_, err = db.Prepare("post_list_asc_flat_since", `
		SELECT author, created, forum, id, message, thread, parent
		FROM posts
		WHERE thread = $1 AND id > $3
		ORDER BY id ASC
		LIMIT $2`)
	if err != nil { log.Fatal(err) }

	_, err = db.Prepare("post_list_asc_flat", `
		SELECT author, created, forum, id, message, thread, parent
		FROM posts
		WHERE thread = $1
		ORDER BY id ASC
		LIMIT $2`)
	if err != nil { log.Fatal(err) }

	_, err = db.Prepare("post_list_asc_tree_since", `
		SELECT author, created, forum, id, message, thread, parent
		FROM posts
		WHERE thread = $1 AND path > (SELECT path FROM posts WHERE id = $3)
		ORDER BY path ASC
		LIMIT $2`)
	if err != nil { log.Fatal(err) }

	_, err = db.Prepare("post_list_asc_tree", `
		SELECT author, created, forum, id, message, thread, parent
		FROM posts
		WHERE thread = $1
		ORDER BY path ASC
		LIMIT $2`)
	if err != nil { log.Fatal(err) }

	_, err = db.Prepare("post_list_asc_parent_tree_since", `
		SELECT author, created, forum, id, message, thread, parent
		FROM posts
		WHERE path[2] IN (
			SELECT id FROM posts
			WHERE thread = $1 AND parent = 0 AND path[2] > (SELECT path[2] FROM posts WHERE id = $3)
			ORDER BY id ASC
			LIMIT $2
		)
		ORDER BY path ASC`)
	if err != nil { log.Fatal(err) }

	_, err = db.Prepare("post_list_asc_parent_tree", `
		SELECT author, created, forum, id, message, thread, parent
		FROM posts
		WHERE path[2] IN (
			SELECT id FROM posts
			WHERE thread = $1 AND parent = 0
			ORDER BY id ASC
			LIMIT $2
		)
		ORDER BY path ASC`)
	if err != nil { log.Fatal(err) }
}
