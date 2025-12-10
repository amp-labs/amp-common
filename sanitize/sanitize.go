// Copyright 2015 Volker Dobler.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package sanitize contains functions to sanitize filenames.
package sanitize

import (
	"bytes"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// commonCharacterReplacements contains URL safe replacements for some
// unsuitable -- but common -- characters in filenames.
var commonCharacterReplacements = map[rune]string{ //nolint:gochecknoglobals
	'ä': "ae", 'Ä': "Ae", 'ö': "oe", 'Ö': "Oe",
	'ü': "ue", 'Ü': "Ue", 'ß': "ss", 'ç': "c",
	'&': "_and_", '+': "_plus_", '@': "_at_",
	'€': "Euro", '£': "Pound", '$': "Dollar", '¥': "Yen",
}

// FileName produces something resembling name but being
// suitable as a filename.
func FileName(name string) string {
	// Eradicate sick characters and perform common replacements.
	if len(name) == 0 {
		return ""
	}

	buf := bytes.Buffer{}

	for _, runeVal := range name {
		switch runeVal {
		// Some characters just do not belong into a filename.
		// Several of these characters are forbidden in Windows, others
		// require quoting in normal shells and the rest is disliked
		// by me. Note that '&' will be replaced by "_and_" and not
		// just dropped.
		case '"', ':', '/', '\\', '(', ')', '?', '*', '\n', '\t', '\r',
			' ', '{', '|', '}', '[', '¦', ']', '!', '#', '%', '<',
			'>', '~', '^', '\'', '`', '°', '§':
			buf.WriteRune('_')
		default:
			buf.WriteRune(runeVal)
		}
	}

	name = buf.String()
	buf.Reset()

	for _, r := range name {
		if repl, ok := commonCharacterReplacements[r]; ok {
			buf.WriteString(repl)
		} else {
			buf.WriteRune(r)
		}
	}

	// Remove accents (combining marks).
	nfd := norm.NFD.String(buf.String())
	buf.Reset()

	for _, r := range nfd {
		if unicode.IsMark(r) {
			continue
		}

		buf.WriteRune(r)
	}

	// Keep only printable ASCII.
	name = buf.String()

	buf.Reset()

	for _, r := range name {
		if r < 32 || r > 126 {
			buf.WriteRune('_')
		} else {
			buf.WriteRune(r)
		}
	}

	name = buf.String()

	// Collapse multiple _ to a single one.
	for strings.Contains(name, "__") {
		name = strings.ReplaceAll(name, "__", "_")
	}

	// trim leading and trailing -
	if name[0] == '-' {
		name = name[1:]
	}

	if l := len(name); l > 1 && name[l-1] == '-' {
		name = name[:len(name)-1]
	}

	return name
}
