package nanostore

// AsTestStore converts a Store to a TestStore if possible
// Returns nil if the store doesn't support testing features
func AsTestStore(s Store) TestStore {
	if ts, ok := s.(TestStore); ok {
		return ts
	}
	return nil
}
