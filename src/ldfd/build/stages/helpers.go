package stages

// containsCat checks if a component's categories slice contains the given category.
func containsCat(categories []string, cat string) bool {
	for _, c := range categories {
		if c == cat {
			return true
		}
	}
	return false
}
