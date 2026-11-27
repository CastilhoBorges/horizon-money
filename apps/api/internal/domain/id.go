package domain

import (
	"encoding/hex"
	"strings"
)

// ID identifies a stored record: a UUID the database generates (UUIDv7). Clients treat it as opaque.
// The zero value means "no id".
type ID [16]byte

// idLength is the canonical form's length: 32 hex digits and 4 dashes (8-4-4-4-12).
const idLength = 36

// ParseID reads the canonical 8-4-4-4-12 form in either letter case. Anything else, and the all-zero
// UUID, is refused with id.invalid, so a malformed id never reaches the database.
func ParseID(s string) (ID, error) {
	if len(s) != idLength || s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return ID{}, ErrInvalidID
	}
	var id ID
	if _, err := hex.Decode(id[:], []byte(strings.ReplaceAll(s, "-", ""))); err != nil || id.IsZero() {
		return ID{}, ErrInvalidID
	}
	return id, nil
}

// String writes the canonical lower-case form.
func (id ID) String() string {
	h := hex.EncodeToString(id[:])
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
}

// IsZero reports whether id is the zero ID.
func (id ID) IsZero() bool { return id == ID{} }
