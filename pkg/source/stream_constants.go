package source

// StreamSignalBufferSize is the capacity of the per-stream, coalesced
// "new data available" notify channels used by the events/command-history/
// links ring buffers. It only ever needs to hold a single pending signal
// (sends are non-blocking and coalesce - see e.g. getEventListener), so 1
// would technically suffice; a small buffer just gives a little slack
// against a scheduling hiccup on the consumer side without ever growing
// unbounded.
const StreamSignalBufferSize = 16

// ParameterRingCapacity is the number of most-recent values retained per
// parameter in its shared ring buffer (see ParameterDemand.Ring). It must
// be large enough to cover the maximum number of values expected to arrive
// for a single parameter between two consecutive drains of its
// slowest-polling stream (RunParameterStream's ticker can be as long as
// 30s - see getStreamTickerInterval) - if a stream falls behind by more
// than this many values, the oldest ones it wanted are already
// overwritten and get reported/dropped instead of delivered.
//
// 1024 covers a sustained ~34 values/sec for the full 30s worst case,
// comfortably above realistic Yamcs telemetry rates (typically well under
// 10Hz per parameter). Every subscribed parameter keeps its own ring
// permanently allocated for as long as it has at least one stream, so this
// constant is a memory/safety-margin trade-off multiplied by parameter
// count, not just a per-stream cost - keep it as small as realistic
// traffic allows rather than defaulting to a very generous number "just in
// case". If production logs ever show the "fell behind its ring buffer
// capacity" warning under normal use, raise this rather than adding
// runtime resizing.
const ParameterRingCapacity = 1024

// BroadcastRingCapacity is the shared ring buffer capacity used for
// instance-wide broadcast stream types (events, command history, links) -
// data that all of an endpoint's stream paths for that type observe
// identically, as opposed to parameters, which are keyed per-parameter.
const BroadcastRingCapacity = 1024
