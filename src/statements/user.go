package statements

import (
	"github.com/jackc/pgx"
)

func UserPrepare(db *pgx.ConnPool) error {
	_, err := db.Prepare("user_get_by_nickname", `
        SELECT nickname, fullname, about, email
        FROM users
        WHERE nickname = $1
        LIMIT 1`)
	if err != nil {
		return err
	}

	return nil
}
