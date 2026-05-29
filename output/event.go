package output

// EventKind enumerates the platform-neutral input event kinds delivered by a
// [LiveSurface].
type EventKind uint8

// Event kinds delivered by a [LiveSurface].
const (
	EventResize EventKind = iota
	EventPointerMove
	EventPointerDown
	EventPointerUp
	EventScroll
	EventKey
	EventClose
)

// Event is the platform-neutral input event. output/window translates OS events
// into it; output/web translates DOM events into it. The Session and Controller
// see only this type — never a platform-specific event.
type Event struct {
	Kind      EventKind
	X, Y      float64 // logical coordinates
	DX, DY    float64 // scroll / drag deltas
	Buttons   uint8
	Key       string
	Modifiers uint8
}
