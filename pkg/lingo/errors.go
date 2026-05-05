package lingo

import "errors"

var ErrInvalidSwitcher = errors.New("switcher must use comma-separated target:path pairs")
