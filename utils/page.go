package utils

// Page limits for list endpoints. A client-supplied limit is never passed to the
// database as is: a negative one means "no limit" to gorm and SQLite, and a large one
// makes a single request read a whole table.
const (
	MaxPageLimit = 100
	// Endpoints that serve whole series or all pairs, e.g. K-line and TVL charts.
	MaxSeriesLimit = 1000
)

// PageLimit bounds limit to [0, max]. A negative limit asks for everything and gets max.
func PageLimit[T ~int | ~int64](limit, max T) T {
	if limit < 0 || limit > max {
		return max
	}
	return limit
}

// PageOffset turns a negative offset, which the databases disagree on, into 0.
func PageOffset[T ~int | ~int64](offset T) T {
	if offset < 0 {
		return 0
	}
	return offset
}
