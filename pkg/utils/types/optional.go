package types

type Optional[T any] struct {
	value    *T
	hasValue bool
}

// OptionalFrom creates an empty Optional, with no value set.
func OptionalOfNil[T any]() *Optional[T] {
	return &Optional[T]{}
}

func OptionalOf[T any](value T) *Optional[T] {
	op := &Optional[T]{}
	op.Set(value)
	return op
}

// Empty checks if the Optional contains no value.
func (optional *Optional[T]) Empty() bool {
	return !optional.hasValue
}

// IsPresent checks if the Optional contains a value.
func (optional *Optional[T]) IsPresent() bool {
	return optional.hasValue
}

// Get retrieves the value if present. Returns the value and a boolean indicating presence.
func (optional *Optional[T]) Get() T {
	var zeroValue T // zero value of type T
	if optional.hasValue {
		return *optional.value
	}
	return zeroValue
}

// GetOrPanic retrieves the value if present. Panics if no value is set.
func (optional *Optional[T]) GetOrPanic() T {
	if optional.hasValue {
		return *optional.value
	}
	panic("Optional has no value")
}

// GetOr retrieves the value if present. Returns the fallback value if no value is set.
func (optional *Optional[T]) GetOr(fallback T) T {
	if optional.hasValue {
		return *optional.value
	}
	return fallback
}

// Set assigns a value to the Optional, marking it as having a value.
func (optional *Optional[T]) Set(value T) {
	// Create a new value on the heap and assign it to the Optional
	optional.value = new(T)
	*optional.value = value
	optional.hasValue = true
}

// Clear resets the Optional, marking it as having no value.
func (optional *Optional[T]) Clear() {
	optional.value = nil // Important to reset the pointer
	optional.hasValue = false
}
