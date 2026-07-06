package sitemap

// authoringFormInputsContain is a sitemap-local duplicate of
// authoring.authoringFormInputsContain (mutations_test.go), needed by
// sitemap_view_test.go. It mirrors the root package's now-removed
// authoring_test_helpers_test.go, which carried the same duplicate while
// sitemap_view_test.go stayed at root pending this slice. It is small
// enough that duplicating it here is simpler and safer than exporting a
// test-only helper across the package boundary.
func authoringFormInputsContain(inputs []map[string]string, name, value string) bool {
	for _, input := range inputs {
		if input["name"] == name && input["value"] == value {
			return true
		}
	}
	return false
}
