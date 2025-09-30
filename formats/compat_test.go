package formats

// Helper functions for tests that use the old signatures
// These allow existing tests to continue working while we add new metadata tests

func serializeWithoutMetadata(format *DocumentFormat, title, content string) string {
	return format.Serialize(title, content, nil)
}

func deserializeWithoutMetadata(format *DocumentFormat, document string) (string, string, error) {
	title, content, _, err := format.Deserialize(document)
	return title, content, err
}
