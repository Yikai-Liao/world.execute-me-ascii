// Package assets contains the metadata shared by both players.
package assets

import "embed"

// FS embeds the original subtitle and spectrum data. A local release build can
// also include bundled/song.mp3; source builds use --audio instead.
//
//go:embed lyrics.json spectrum.json config.json bundled/*
var FS embed.FS
