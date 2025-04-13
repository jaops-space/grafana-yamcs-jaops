package types

// Set is a generic type representing a set of values of type T.
// It uses a map internally for efficient storage and lookups.
type Set[T comparable] map[T]struct{}

// NewSet creates and returns a new empty Set.
func NewSet[T comparable]() Set[T] {
	return make(Set[T])
}

// Add inserts one or more values into the set.
// If the value already exists, it is ignored (sets do not contain duplicates).
func (s Set[T]) Add(values ...T) {
	for _, value := range values {
		s[value] = struct{}{}
	}
}

// Remove deletes one or more values from the set.
// If a value does not exist in the set, it is simply ignored.
func (s Set[T]) Remove(values ...T) {
	for _, value := range values {
		delete(s, value)
	}
}

// Exists checks whether a value is present in the set.
// It returns true if the value exists, and false otherwise.
func (s Set[T]) Exists(value T) bool {
	_, found := s[value]
	return found
}

// Size returns the number of elements in the set.
func (s Set[T]) Size() int {
	return len(s)
}

// Clear removes all elements from the set, making it empty.
func (s Set[T]) Clear() {
	for key := range s {
		delete(s, key)
	}
}
