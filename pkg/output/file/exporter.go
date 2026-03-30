package file

import (
	"fmt"
	"path/filepath"

	"github.com/TuSKan/ggplot/pkg/output"
	"github.com/gogpu/gg"
)

// FileExporter implements output.Exporter evaluating artifacts through standard gg recording
// mechanisms before serializing to disk.
type FileExporter struct{}

// NewFileExporter constructs a new file-based rendering backend.
func NewFileExporter() *FileExporter {
	return &FileExporter{}
}

// Export evaluates the output tree and saves the compiled representation to disk.
func (e *FileExporter) Export(o *output.Output, filename string) error {
	// Replicating basic gg drawing pipeline for saving files
	if o.Draw == nil {
		return fmt.Errorf("no draw function provided in output")
	}

	ctx := gg.NewContext(o.Width, o.Height)

	if o.Theme.Background != nil {
		ctx.ClearWithColor(gg.FromColor(o.Theme.Background))
	}
	o.Draw(ctx)

	ext := filepath.Ext(filename)
	if ext == ".png" {
		return ctx.SavePNG(filename)
	}

	// This assumes backend-specific registrations handle formats
	// via standard graphics contexts, extending this based on 'gg' capabilities.
	return fmt.Errorf("unsupported export file extension: %s", ext)
}
