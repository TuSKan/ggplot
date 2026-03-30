package fonts

import (
	"golang.org/x/image/font/sfnt"
	"strings"
)

// parseFontFile loads a byte slice against the true-type schema evaluating collections.
func parseFontFile(path string, data []byte) ([]Font, error) {
	if coll, err := sfnt.ParseCollection(data); err == nil {
		var fonts []Font

		n := coll.NumFonts()
		for i := 0; i < n; i++ {
			f, err := coll.Font(i)
			if err != nil {
				continue
			}

			font, err := extractFontMetadata(path, f, i)
			if err == nil {
				fonts = append(fonts, *font)
			}
		}
		return fonts, nil
	}

	f, err := sfnt.Parse(data)
	if err != nil {
		return nil, ErrInvalidFontData
	}

	font, err := extractFontMetadata(path, f, 0)
	if err != nil {
		return nil, err
	}

	return []Font{*font}, nil
}

// Iterates across multiple TrueType label tables computing implicit rendering tags.
func extractFontMetadata(path string, f *sfnt.Font, index int) (*Font, error) {
	var buf sfnt.Buffer

	family, _ := f.Name(&buf, sfnt.NameIDFamily)
	subfamily, _ := f.Name(&buf, sfnt.NameIDSubfamily)
	fullName, _ := f.Name(&buf, sfnt.NameIDFull)

	comb := strings.ToLower(family + " " + subfamily + " " + fullName)

	// Broad system level extraction parsing flags.
	isMono := strings.Contains(comb, "mono") ||
		strings.Contains(comb, "console") ||
		strings.Contains(comb, "courier") ||
		strings.Contains(comb, "typewriter") ||
		strings.Contains(comb, "menlo") ||
		strings.Contains(comb, "cascadia code")

	isSym := strings.Contains(comb, "emoji") ||
		strings.Contains(comb, "symbol") ||
		strings.Contains(comb, "math") ||
		strings.Contains(comb, "icons") ||
		strings.Contains(comb, "dingbats")

	stretch := "normal"
	if strings.Contains(comb, "condensed") || strings.Contains(comb, "compressed") || strings.Contains(comb, "narrow") {
		stretch = "condensed"
	} else if strings.Contains(comb, "expanded") || strings.Contains(comb, "extended") || strings.Contains(comb, "wide") {
		stretch = "expanded"
	}

	return &Font{
		Path:        path,
		Index:       index,
		Family:      normalizeFamily(family),
		Subfamily:   strings.TrimSpace(subfamily),
		FullName:    strings.TrimSpace(fullName),
		Weight:      parseWeight(subfamily),
		Style:       parseStyle(subfamily),
		Stretch:     stretch,
		IsMonospace: isMono,
		IsSymbol:    isSym,
	}, nil
}
