package api

// nilIfStringEmpty returns a nil if the string is 0 length and the string pointer otherwise
func nilIfStringEmpty(s *string) *string {
	if s == nil || len(*s) == 0 {
		return nil
	}
	return s
}
