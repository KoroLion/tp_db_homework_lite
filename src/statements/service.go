package statements

import (
	"github.com/jackc/pgx"
)

func ServicePrepare(db *pgx.ConnPool) error {
	_, err := db.Prepare("service_status", `
        SELECT users, forums, threads, posts FROM status LIMIT 1
    `)
	if err != nil {
		return err
	}

	return nil
}
