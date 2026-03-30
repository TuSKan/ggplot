package ui_test

import (
	"github.com/TuSKan/ggplot/pkg/output/ui"
	"testing"
)

func TestGPUWindowPresenter_Construct(t *testing.T) {
	p := ui.NewGPUWindowPresenter()
	if p == nil {
		t.Fatal("NewGPUWindowPresenter returned nil")
	}
}
