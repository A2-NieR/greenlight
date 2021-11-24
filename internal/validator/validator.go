package validator

import "regexp"

// EmailRX regexp for email validation
var EmailRX = regexp.MustCompile("^\\w+([-+.']\\w+)*@\\w+([-.]\\w+)*\\.\\w+([-.]\\w+)*$")

// Validator type containing a map of validation errors
type Validator struct {
	Errors map[string]string
}

// New helper creates a new Validator instance with empty errors map
func New() *Validator {
	return &Validator{Errors: make(map[string]string)}
}

// Valid returns true if errors map doesn't contain entries
func (v *Validator) Valid() bool {
	return len(v.Errors) == 0
}

// AddError adds an error message to the map if no entry exists for given key
func (v *Validator) AddError(key, message string) {
	if _, exists := v.Errors[key]; !exists {
		v.Errors[key] = message
	}
}

// Check adds an error message to the map if validation check is not ok
func (v *Validator) Check(ok bool, key, message string) {
	if !ok {
		v.AddError(key, message)
	}
}

// In returns true if specific value is in list of strings
func In(value string, list ...string) bool {
	for i := range list {
		if value == list[i] {
			return true
		}
	}

	return false
}

// Matches returns true if string value matches specific regexp pattern
func Matches(value string, rx *regexp.Regexp) bool {
	return rx.MatchString(value)
}

// Unique returns true if all string values in slice are unique
func Unique(values []string) bool {
	uniqueValues := make(map[string]bool)

	for _, value := range values {
		uniqueValues[value] = true
	}

	return len(values) == len(uniqueValues)
}
