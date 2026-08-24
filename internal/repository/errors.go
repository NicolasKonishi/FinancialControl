package repository

import "errors"

// ErrNotFound is returned when a row does not exist.
var ErrNotFound = errors.New("not found")
