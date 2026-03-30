package fonts

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// DefaultDirs provides the universal logical starting OS-specific standard hierarchies.
func DefaultDirs() []string {
	switch runtime.GOOS {
	case "windows":
		return []string{`C:\Windows\Fonts`}
	case "darwin":
		return []string{
			"/System/Library/Fonts",
			"/Library/Fonts",
			filepath.Join(os.Getenv("HOME"), "Library/Fonts"),
		}
	default:
		return []string{
			"/usr/share/fonts",
			"/usr/local/share/fonts",
			filepath.Join(os.Getenv("HOME"), ".fonts"),
		}
	}
}

// DiscoverFonts iteratively paths recursively searching for supported binary format schemas parsing.
func DiscoverFonts(dirs []string) ([]Font, error) {
	var available []Font

	for _, dir := range dirs {
		filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d == nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}

			ext := strings.ToLower(filepath.Ext(path))
			if ext != ".ttf" && ext != ".otf" && ext != ".ttc" && ext != ".otc" {
				return nil
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			fonts, err := parseFontFile(path, data)
			if err == nil {
				available = append(available, fonts...)
			}

			return nil
		})
	}

	return available, nil
}
