package fonts

import "errors"

// ErrFontNotFound is returned when no suitable font or fallback can be matched.
var ErrFontNotFound = errors.New("font not found")

// ErrInvalidFontData is returned when a font file cannot be parsed.
var ErrInvalidFontData = errors.New("invalid font data")
