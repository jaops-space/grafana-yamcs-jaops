package types

import "sync/atomic"

// Ring is a fixed-capacity, single-producer/multi-consumer circular buffer.
//
// Push must only ever be called by one goroutine at a time (never
// concurrently with itself) - in this codebase that's always the
// WebSocket's single read-loop goroutine for a given subscription type.
// Any number of consumer goroutines may call DrainSince concurrently with
// each other and with Push, each maintaining its own cursor, with zero
// locking on either side of the hot path: the backing array is
// preallocated once and never reallocated/resized, and the write-position
// counter is only ever published/observed via atomics, so there is no
// multi-word slice header that could be read half-updated (unlike a
// growable slice, whose pointer/len/cap can change together on append and
// would require a mutex to publish safely).
type Ring[T any] struct {
	cap     uint64
	slots   []T
	written atomic.Uint64 // total values ever pushed (write cursor)
}

// NewRing creates a ring buffer that retains up to capacity of the most
// recently pushed values. Size capacity for the maximum number of values
// you expect to arrive between two consecutive drains of the slowest
// consumer - if a consumer falls behind by more than capacity, the oldest
// values it wanted have already been physically overwritten; DrainSince
// detects this and resyncs instead of returning stale data (see dropped
// return value).
func NewRing[T any](capacity int) *Ring[T] {
	if capacity < 1 {
		capacity = 1
	}
	return &Ring[T]{cap: uint64(capacity), slots: make([]T, capacity)}
}

// Push stores v as the next value in the ring. Callers must serialize their
// own calls to Push (single producer only); it is not safe to call Push
// concurrently from multiple goroutines.
func (r *Ring[T]) Push(v T) {
	idx := r.written.Load()
	r.slots[idx%r.cap] = v
	r.written.Store(idx + 1)
}

// Cursor returns the ring's current write position - the value a new
// consumer should start its own cursor at if it should only observe values
// pushed after it started watching, not any pre-existing backlog.
func (r *Ring[T]) Cursor() uint64 {
	return r.written.Load()
}

// DrainSince returns every value pushed since cursor (exclusive), along
// with the new cursor the caller should pass on its next call. If the
// caller has fallen behind by more than the ring's capacity, cursor is
// resynced to the oldest value still retained and dropped is reported as
// true, instead of returning stale/overwritten slots.
func (r *Ring[T]) DrainSince(cursor uint64) (values []T, newCursor uint64, dropped bool) {
	total := r.written.Load()
	if total < cursor {
		// Should not normally happen (e.g. a cursor from a different ring
		// instance); guard against an underflow that would otherwise wrap
		// total-cursor into an enormous uint64.
		cursor = total
	}
	if total-cursor > r.cap {
		cursor = total - r.cap
		dropped = true
	}
	if total == cursor {
		return nil, total, dropped
	}
	values = make([]T, 0, total-cursor)
	for i := cursor; i < total; i++ {
		values = append(values, r.slots[i%r.cap])
	}
	return values, total, dropped
}
