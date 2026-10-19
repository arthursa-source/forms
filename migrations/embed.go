// Package migrations guarda os arquivos .sql versionados e os embute no
// binário (go:embed), para que `server migrate up` funcione em qualquer
// máquina sem depender de caminhos relativos.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
