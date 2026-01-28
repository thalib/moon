// Package ulid provides ULID generation and validation functionality.
// ULIDs (Universally Unique Lexicographically Sortable Identifiers) are
// 26-character, URL-safe, base32-encoded strings that are sortable by creation time.
package ulid

import (
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"
)

var (
	// ErrInvalidULID indicates that a ULID string is malformed or invalid
	ErrInvalidULID = errors.New("invalid ULID format")
)

// Generate creates a new ULID using the current timestamp and secure random data
func Generate() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()
}

// GenerateWithTime creates a new ULID using the specified timestamp and secure random data
func GenerateWithTime(t time.Time) string {
	return ulid.MustNew(ulid.Timestamp(t), rand.Reader).String()
}

// Validate checks if a string is a valid ULID format (26 characters, base32 encoded)
func Validate(str string) error {
	if len(str) != 26 {
		return fmt.Errorf("%w: expected 26 characters, got %d", ErrInvalidULID, len(str))
	}

	// Try to parse the ULID
	if _, err := ulid.Parse(str); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidULID, err)
	}

	return nil
}

// IsValid returns true if the string is a valid ULID, false otherwise
func IsValid(str string) bool {
	return Validate(str) == nil
}

// Parse parses a ULID string and returns the underlying ULID value
func Parse(str string) (ulid.ULID, error) {
	id, err := ulid.Parse(str)
	if err != nil {
		return ulid.ULID{}, fmt.Errorf("%w: %v", ErrInvalidULID, err)
	}
	return id, nil
}

// Time extracts the timestamp from a ULID string
func Time(str string) (time.Time, error) {
	id, err := Parse(str)
	if err != nil {
		return time.Time{}, err
	}
	return ulid.Time(id.Time()), nil
}

// Compare compares two ULID strings lexicographically
// Returns -1 if a < b, 0 if a == b, 1 if a > b
func Compare(a, b string) (int, error) {
	idA, err := Parse(a)
	if err != nil {
		return 0, err
	}

	idB, err := Parse(b)
	if err != nil {
		return 0, err
	}

	return idA.Compare(idB), nil
}
