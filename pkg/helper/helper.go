// Package helper holds small, generic helpers with no business meaning
// that are reused across modules and layers.
package helper

// Ptr returns a pointer to the given value, useful for optional fields.
func Ptr[T any](v T) *T {
	return &v
}

// IsZero reports whether v is the zero value of its type.
func IsZero[T comparable](v T) bool {
	var zero T
	return v == zero
}

// Coalesce returns the first non-zero value, or the zero value if all are zero.
func Coalesce[T comparable](values ...T) T {
	var zero T
	for _, v := range values {
		if v != zero {
			return v
		}
	}
	return zero
}
