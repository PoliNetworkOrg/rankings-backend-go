package utils

// MergeUnique is a generic function that takes two slices of comparable type T
// and returns a new slice with all unique values from both.
// It preserves the order of appearance (first from 'a', then from 'b' for new elements).
// Time: O(n + m), Space: O(n + m) where n=len(a), m=len(b).
// Uses a map[T]struct{} for efficient uniqueness checks (Go's equivalent to a set).
func MergeUnique[T comparable](a, b []T) []T {
	seen := make(map[T]struct{})
	result := make([]T, 0, len(a)+len(b))

	// Add unique from first slice
	for _, item := range a {
		if _, exists := seen[item]; !exists {
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}

	// Add unique from second slice
	for _, item := range b {
		if _, exists := seen[item]; !exists {
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}

	return result
}
