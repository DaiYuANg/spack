// Package view embeds HTML templates used by the server.
package view

import "embed"

//go:embed *.html
var View embed.FS
