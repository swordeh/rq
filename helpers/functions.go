package helpers

// Contains iterates through a slice to check for the presence of str
func Contains(slice *[]string, str string) bool {
	for _, s := range *slice {
		if s == str {
			return true
		}
	}
	return false
}
