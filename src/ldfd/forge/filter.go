package forge

import (
	"strings"
)

// VersionFilter handles glob-based version filtering
type VersionFilter struct {
	includePatterns []string
	excludePatterns []string
}

// ParseVersionFilter parses a comma-separated filter string into a VersionFilter
// Filter syntax:
//   - "pattern" or "+pattern" - include versions matching the glob pattern
//   - "!pattern" - exclude versions matching the glob pattern
//   - Patterns are applied in order: includes first, then excludes
//   - If no include patterns, all versions are included by default
//
// Examples:
//   - "!*-rc*,!*alpha*,!*beta*" - exclude RC, alpha, and beta versions
//   - "6.*" - only include versions starting with 6.
//   - "6.*,!*-rc*" - include 6.x versions, excluding RCs
func ParseVersionFilter(filterStr string) *VersionFilter {
	vf := &VersionFilter{
		includePatterns: make([]string, 0),
		excludePatterns: make([]string, 0),
	}

	if filterStr == "" {
		return vf
	}

	// Split by comma and trim whitespace
	parts := strings.Split(filterStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.HasPrefix(part, "!") {
			// Exclude pattern
			pattern := strings.TrimPrefix(part, "!")
			if pattern != "" {
				vf.excludePatterns = append(vf.excludePatterns, pattern)
			}
		} else if strings.HasPrefix(part, "+") {
			// Explicit include pattern
			pattern := strings.TrimPrefix(part, "+")
			if pattern != "" {
				vf.includePatterns = append(vf.includePatterns, pattern)
			}
		} else {
			// Implicit include pattern
			vf.includePatterns = append(vf.includePatterns, part)
		}
	}

	return vf
}

// Matches checks if a version string matches the filter
// Returns true if the version should be included, false if excluded
func (vf *VersionFilter) Matches(version string) bool {
	// If there are include patterns, version must match at least one
	if len(vf.includePatterns) > 0 {
		matched := false
		for _, pattern := range vf.includePatterns {
			if globMatch(pattern, version) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check exclude patterns - if any match, exclude the version
	for _, pattern := range vf.excludePatterns {
		if globMatch(pattern, version) {
			return false
		}
	}

	return true
}

// IsEmpty returns true if the filter has no patterns
func (vf *VersionFilter) IsEmpty() bool {
	return len(vf.includePatterns) == 0 && len(vf.excludePatterns) == 0
}

// String returns the filter as a comma-separated string
func (vf *VersionFilter) String() string {
	parts := make([]string, 0, len(vf.includePatterns)+len(vf.excludePatterns))

	for _, p := range vf.includePatterns {
		parts = append(parts, p)
	}
	for _, p := range vf.excludePatterns {
		parts = append(parts, "!"+p)
	}

	return strings.Join(parts, ",")
}

// FilterVersions filters a slice of versions using this filter
func (vf *VersionFilter) FilterVersions(versions []DiscoveredVersion) []DiscoveredVersion {
	if vf.IsEmpty() {
		return versions
	}

	result := make([]DiscoveredVersion, 0, len(versions))
	for _, v := range versions {
		if vf.Matches(v.Version) {
			result = append(result, v)
		}
	}
	return result
}

// globMatch performs glob pattern matching with support for:
//   - * matches any sequence of characters (including empty)
//   - ? matches any single character
//
// The matching is case-insensitive for convenience
func globMatch(pattern, str string) bool {
	// Case-insensitive matching
	pattern = strings.ToLower(pattern)
	str = strings.ToLower(str)

	return globMatchInternal(pattern, str)
}

// globMatchInternal is the recursive glob matching implementation
func globMatchInternal(pattern, str string) bool {
	for len(pattern) > 0 {
		switch pattern[0] {
		case '*':
			// Skip consecutive stars
			for len(pattern) > 0 && pattern[0] == '*' {
				pattern = pattern[1:]
			}

			// Trailing * matches everything
			if len(pattern) == 0 {
				return true
			}

			// Try matching the rest of the pattern at each position
			for i := 0; i <= len(str); i++ {
				if globMatchInternal(pattern, str[i:]) {
					return true
				}
			}
			return false

		case '?':
			// ? matches exactly one character
			if len(str) == 0 {
				return false
			}
			pattern = pattern[1:]
			str = str[1:]

		default:
			// Regular character must match exactly
			if len(str) == 0 || pattern[0] != str[0] {
				return false
			}
			pattern = pattern[1:]
			str = str[1:]
		}
	}

	// Pattern exhausted, string must also be exhausted
	return len(str) == 0
}

// FilterResult contains the result of applying a filter to a version
type FilterResult struct {
	Version  string `json:"version"`
	Included bool   `json:"included"`
	Reason   string `json:"reason,omitempty"` // Which pattern matched/excluded it
}

// FilterWithReasons filters versions and returns detailed results
func (vf *VersionFilter) FilterWithReasons(versions []string) []FilterResult {
	results := make([]FilterResult, 0, len(versions))

	for _, version := range versions {
		result := FilterResult{
			Version:  version,
			Included: true,
		}

		// Check include patterns
		if len(vf.includePatterns) > 0 {
			matched := false
			for _, pattern := range vf.includePatterns {
				if globMatch(pattern, version) {
					matched = true
					result.Reason = "matches " + pattern
					break
				}
			}
			if !matched {
				result.Included = false
				result.Reason = "no include pattern matched"
			}
		}

		// Check exclude patterns (only if still included)
		if result.Included {
			for _, pattern := range vf.excludePatterns {
				if globMatch(pattern, version) {
					result.Included = false
					result.Reason = "excluded by !" + pattern
					break
				}
			}
		}

		results = append(results, result)
	}

	return results
}

// ValidateFilter checks if a filter string is syntactically valid
func ValidateFilter(filterStr string) error {
	// Parse and ensure it doesn't panic
	_ = ParseVersionFilter(filterStr)
	return nil
}

// CommonFilters provides commonly used filter presets
var CommonFilters = map[string]string{
	"stable-only":   "!*-rc*,!*alpha*,!*beta*,!*-dev*,!*-pre*,!*-snapshot*,!*-nightly*",
	"no-rc":         "!*-rc*",
	"lts-only":      "6.12.*,6.6.*,6.1.*,5.15.*,5.10.*,5.4.*",
	"latest-major":  "6.*",
	"kernel-stable": "!*-rc*,!next-*",
	"kernel-lts":    "6.12.*,6.6.*,6.1.*,5.15.*,5.10.*,5.4.*,4.19.*,4.14.*,!*-rc*",
}

// GetCommonFilter returns a common filter by name
func GetCommonFilter(name string) string {
	if filter, ok := CommonFilters[name]; ok {
		return filter
	}
	return ""
}
