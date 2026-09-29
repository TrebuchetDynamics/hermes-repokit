package verify

import "regexp"

var platformName = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)
