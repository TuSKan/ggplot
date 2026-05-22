package scale

import (
	"math"
	"time"

	"github.com/TuSKan/ggplot/dataset"
)

// DateUnit identifies the granularity for datetime tick generation.
type DateUnit int

const (
	// DateUnitSecond places ticks at second boundaries.
	DateUnitSecond DateUnit = iota
	// DateUnitMinute places ticks at minute boundaries.
	DateUnitMinute
	// DateUnitHour places ticks at hour boundaries.
	DateUnitHour
	// DateUnitDay places ticks at day boundaries.
	DateUnitDay
	// DateUnitWeek places ticks at week boundaries.
	DateUnitWeek
	// DateUnitMonth places ticks at month boundaries.
	DateUnitMonth
	// DateUnitYear places ticks at year boundaries.
	DateUnitYear
)

// Nanosecond constants for time span calculations.
const (
	nsPerSecond = 1e9
	nsPerMinute = 60 * nsPerSecond
	nsPerHour   = 60 * nsPerMinute
	nsPerDay    = 24 * nsPerHour
)

// DateTimeScale is a continuous scale for temporal data (timestamps, dates,
// times-of-day). It auto-detects tick granularity from the data span and
// formats labels using Go's time package.
//
// Train accepts Column[int64] with DType ∈ {DTypeTimestamp, DTypeDate, DTypeTime}.
// For DTypeDate columns, values are days since epoch; for DTypeTimestamp/DTypeTime,
// values are nanoseconds.
type DateTimeScale struct {
	domain // reuses existing min/max tracking

	tz     *time.Location
	unit   DateUnit // auto-detected granularity
	layout string   // format string for labels
	isDate bool     // true when trained from DTypeDate (values are days, not ns)
}

// NewDateTime returns a DateTime scale using the local timezone.
func NewDateTime() Scale {
	return &DateTimeScale{tz: time.Local}
}

// NewDateTimeIn returns a DateTime scale using the specified timezone.
func NewDateTimeIn(loc *time.Location) Scale {
	return &DateTimeScale{tz: loc}
}

// Train updates the scale domain from temporal column data.
func (s *DateTimeScale) Train(col dataset.AnyColumn) error {
	switch col.DType() { //nolint:exhaustive // only temporal types need isDate flag.
	case dataset.DTypeDate:
		s.isDate = true
	case dataset.DTypeTimestamp, dataset.DTypeTime:
		s.isDate = false
	}

	return s.train(col)
}

// Map transforms a temporal value to a [0, 1] linear fraction.
func (s *DateTimeScale) Map(v float64) float64 {
	if s.max == s.min {
		return 0.5
	}

	return (v - s.min) / (s.max - s.min)
}

// Inverse maps a [0, 1] fraction back to data space.
func (s *DateTimeScale) Inverse(v float64) float64 {
	return s.min + v*(s.max-s.min)
}

// Ticks generates approximately n calendar-aligned tick positions.
func (s *DateTimeScale) Ticks(n int) []float64 {
	if n <= 0 {
		n = 5
	}

	span := s.max - s.min
	if span <= 0 {
		return []float64{s.min}
	}

	// Convert to nanoseconds for granularity selection.
	spanNs := span
	if s.isDate {
		spanNs = span * nsPerDay
	}

	s.unit, s.layout = detectGranularity(spanNs)

	return s.generateTicks(n)
}

// Format converts a temporal value to a display string.
func (s *DateTimeScale) Format(v float64) string {
	t := s.toTime(v)
	if s.layout == "" {
		s.detectFromSpan()
	}

	return t.Format(s.layout)
}

// Bounds returns the trained domain [min, max].
func (s *DateTimeScale) Bounds() (float64, float64) { return s.min, s.max }

// String returns "datetime".
func (s *DateTimeScale) String() string { return "datetime" }

// SetBounds manually overrides the scale domain.
func (s *DateTimeScale) SetBounds(mn, mx float64) {
	s.min = mn
	s.max = mx
	s.trained = true
}

// toTime converts a domain value to a Go time.Time.
func (s *DateTimeScale) toTime(v float64) time.Time {
	if s.isDate {
		// v is days since epoch.
		days := int64(v)
		return time.Unix(days*86400, 0).In(s.tz) //nolint:mnd // seconds-per-day is a domain constant.
	}
	// v is nanoseconds since epoch.
	return time.Unix(0, int64(v)).In(s.tz)
}

// fromTime converts a time.Time to a domain value.
func (s *DateTimeScale) fromTime(t time.Time) float64 {
	if s.isDate {
		return float64(t.Unix() / 86400) //nolint:mnd // seconds-per-day is a domain constant.
	}

	return float64(t.UnixNano())
}

// detectFromSpan sets unit and layout from the current domain span.
func (s *DateTimeScale) detectFromSpan() {
	span := s.max - s.min
	if s.isDate {
		span *= nsPerDay
	}

	s.unit, s.layout = detectGranularity(span)
}

// detectGranularity selects tick granularity and format layout from a span in ns.
func detectGranularity(spanNs float64) (DateUnit, string) {
	switch {
	case spanNs < 2*nsPerMinute:
		return DateUnitSecond, "15:04:05"
	case spanNs < 2*nsPerHour:
		return DateUnitMinute, "15:04"
	case spanNs < 2*nsPerDay:
		return DateUnitHour, "15:04"
	case spanNs < 60*nsPerDay:
		return DateUnitDay, "Jan 2"
	case spanNs < 2*365.25*nsPerDay:
		return DateUnitMonth, "Jan 2006"
	default:
		return DateUnitYear, "2006"
	}
}

// generateTicks produces calendar-aligned tick positions.
func (s *DateTimeScale) generateTicks(n int) []float64 {
	tMin := s.toTime(s.min)
	tMax := s.toTime(s.max)

	var ticks []float64

	switch s.unit {
	case DateUnitSecond:
		ticks = s.ticksByDuration(tMin, tMax, n, []time.Duration{
			time.Second, 2 * time.Second, 5 * time.Second,
			10 * time.Second, 15 * time.Second, 30 * time.Second,
		})
	case DateUnitMinute:
		ticks = s.ticksByDuration(tMin, tMax, n, []time.Duration{
			time.Minute, 2 * time.Minute, 5 * time.Minute,
			10 * time.Minute, 15 * time.Minute, 30 * time.Minute,
		})
	case DateUnitHour:
		ticks = s.ticksByDuration(tMin, tMax, n, []time.Duration{
			time.Hour, 2 * time.Hour, 3 * time.Hour,
			4 * time.Hour, 6 * time.Hour, 12 * time.Hour,
		})
	case DateUnitDay:
		ticks = s.ticksByDuration(tMin, tMax, n, []time.Duration{
			24 * time.Hour, 2 * 24 * time.Hour, 7 * 24 * time.Hour,
		})
	case DateUnitWeek:
		ticks = s.ticksByDuration(tMin, tMax, n, []time.Duration{
			7 * 24 * time.Hour, 14 * 24 * time.Hour,
		})
	case DateUnitMonth:
		ticks = s.ticksByMonth(tMin, tMax, n, []int{1, 2, 3, 6})
	case DateUnitYear:
		ticks = s.ticksByYear(tMin, tMax, n)
	}

	return ticks
}

// ticksByDuration generates ticks at fixed-duration intervals.
func (s *DateTimeScale) ticksByDuration(tMin, tMax time.Time, n int, steps []time.Duration) []float64 {
	span := tMax.Sub(tMin)
	// Pick the step that produces closest to n ticks.
	bestStep := steps[0]
	bestDiff := math.Abs(float64(span/bestStep) - float64(n))

	for _, step := range steps[1:] {
		count := float64(span / step)
		diff := math.Abs(count - float64(n))

		if diff < bestDiff {
			bestStep = step
			bestDiff = diff
		}
	}

	// Snap tMin down to the nearest step boundary.
	startNs := tMin.UnixNano()
	stepNs := int64(bestStep)
	alignedStart := startNs - (startNs % stepNs)

	if alignedStart < startNs {
		alignedStart += stepNs
	}

	var ticks []float64

	for ns := alignedStart; ns <= tMax.UnixNano(); ns += stepNs {
		t := time.Unix(0, ns).In(s.tz)
		ticks = append(ticks, s.fromTime(t))
	}

	return ticks
}

// ticksByMonth generates ticks at month boundaries.
func (s *DateTimeScale) ticksByMonth(tMin, tMax time.Time, n int, monthSteps []int) []float64 {
	// Pick the month step closest to n ticks.
	totalMonths := (tMax.Year()-tMin.Year())*12 + int(tMax.Month()) - int(tMin.Month())

	bestStep := monthSteps[0]
	bestDiff := math.Abs(float64(totalMonths/bestStep) - float64(n))

	for _, step := range monthSteps[1:] {
		count := float64(totalMonths / step)
		diff := math.Abs(count - float64(n))

		if diff < bestDiff {
			bestStep = step
			bestDiff = diff
		}
	}

	// Start at the first of the next aligned month.
	startYear := tMin.Year()
	startMonth := int(tMin.Month())
	// Align to month step.
	startMonth = ((startMonth-1)/bestStep)*bestStep + 1

	var ticks []float64

	for y, m := startYear, startMonth; ; m += bestStep {
		for m > 12 {
			m -= 12
			y++
		}

		t := time.Date(y, time.Month(m), 1, 0, 0, 0, 0, s.tz)
		if t.After(tMax) {
			break
		}

		if !t.Before(tMin) {
			ticks = append(ticks, s.fromTime(t))
		}
	}

	return ticks
}

// ticksByYear generates ticks at year boundaries.
func (s *DateTimeScale) ticksByYear(tMin, tMax time.Time, n int) []float64 {
	yearSpan := tMax.Year() - tMin.Year()
	if yearSpan <= 0 {
		yearSpan = 1
	}

	// Pick a nice year step.
	yearSteps := []int{1, 2, 5, 10, 20, 25, 50, 100}
	bestStep := 1
	bestDiff := math.Abs(float64(yearSpan/1) - float64(n))

	for _, step := range yearSteps {
		if step > yearSpan {
			break
		}

		count := float64(yearSpan / step)
		diff := math.Abs(count - float64(n))

		if diff < bestDiff {
			bestStep = step
			bestDiff = diff
		}
	}

	startYear := (tMin.Year() / bestStep) * bestStep
	if startYear < tMin.Year() {
		startYear += bestStep
	}

	var ticks []float64

	for y := startYear; y <= tMax.Year(); y += bestStep {
		t := time.Date(y, 1, 1, 0, 0, 0, 0, s.tz)
		ticks = append(ticks, s.fromTime(t))
	}

	return ticks
}

// --- Compile-time interface checks ---

var (
	_ Scale        = (*DateTimeScale)(nil)
	_ BoundsSetter = (*DateTimeScale)(nil)
)
