package database

import (
	"database/sql"
	"fmt"

	"github.com/arthursa-source/forms/migrations"
	"github.com/pressly/goose/v3"
)

// Migrate aplica (direction "up") ou desfaz a última (direction "down")
// migration. Os arquivos .sql vêm embutidos no binário.
func Migrate(db *sql.DB, direction string) error {
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	// "." porque o embed.FS tem os .sql na raiz do pacote migrations.
	switch direction {
	case "up":
		return goose.Up(db, ".")
	case "down":
		return goose.Down(db, ".")
	case "status":
		return goose.Status(db, ".")
	default:
		return fmt.Errorf("direção inválida: %q (use up, down ou status)", direction)
	}
}
