// Package migrations embeds the SQL migrations into the binaries, so running
// them needs neither the source tree nor an external tool.
package migrations

import "embed"

//go:embed *.sql
var Files embed.FS
