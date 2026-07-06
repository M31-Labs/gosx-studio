package studio

// authoringFormInputsContain is a root-local duplicate of
// authoring.authoringFormInputsContain (mutations_test.go), needed by
// sitemap_view_test.go (which stays at root pending the sitemap slice). It is
// small enough that duplicating it here is simpler and safer than exporting a
// test-only helper across the package boundary.
func authoringFormInputsContain(inputs []map[string]string, name, value string) bool {
	for _, input := range inputs {
		if input["name"] == name && input["value"] == value {
			return true
		}
	}
	return false
}
