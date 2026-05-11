package colormap

import "errors"

// Sentinel errors for the colormap package.
var (
	// ErrEmptyColor is returned for empty color strings.
	ErrEmptyColor = errors.New("colormap: empty color string")

	// ErrUnknownAlias is returned for unrecognized color aliases.
	ErrUnknownAlias = errors.New("colormap: unknown color alias")

	// ErrParseColor is returned when a color string cannot be parsed.
	ErrParseColor = errors.New("colormap: cannot parse color")

	// ErrMalformedLiteral is returned for malformed color literals.
	ErrMalformedLiteral = errors.New("colormap: malformed color literal")

	// ErrComponentCount is returned when a color has the wrong number of components.
	ErrComponentCount = errors.New("colormap: wrong number of color components")

	// ErrNilCmap is returned when a nil Cmap is registered.
	ErrNilCmap = errors.New("colormap: cannot register nil Cmap")

	// ErrEmptyName is returned for empty colormap names.
	ErrEmptyName = errors.New("colormap: empty name")

	// ErrAlreadyRegistered is returned when a name is already registered.
	ErrAlreadyRegistered = errors.New("colormap: name already registered")

	// ErrNotFound is returned when a colormap is not found.
	ErrNotFound = errors.New("colormap: colormap not found")

	// ErrPositiveRequired is returned when strictly positive values are needed.
	ErrPositiveRequired = errors.New("colormap: strictly positive values required")

	// ErrInvalidRange is returned when a normalization range is invalid.
	ErrInvalidRange = errors.New("colormap: invalid range")

	// ErrNonNumeric is returned for non-numeric column types.
	ErrNonNumeric = errors.New("colormap: non-numeric column type")
)
