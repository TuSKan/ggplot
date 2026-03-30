package output_test

import (
	"github.com/TuSKan/ggplot/pkg/output"
	"testing"
)

// MockPresenter implements the Presenter interface for testing.
type MockPresenter struct {
	CapturedOutput *output.Output
	ErrToReturn    error
}

func (m *MockPresenter) Show(o *output.Output) error {
	m.CapturedOutput = o
	return m.ErrToReturn
}

// MockExporter implements the Exporter interface for testing.
type MockExporter struct {
	CapturedOutput *output.Output
	CapturedFile   string
	ErrToReturn    error
}

func (m *MockExporter) Export(o *output.Output, filename string) error {
	m.CapturedOutput = o
	m.CapturedFile = filename
	return m.ErrToReturn
}

func TestOutput_Show_Delegation(t *testing.T) {
	mockPresenter := &MockPresenter{}
	originalPresenter := output.DefaultPresenter
	defer func() { output.DefaultPresenter = originalPresenter }()

	output.DefaultPresenter = mockPresenter

	out := &output.Output{
		Width:  800,
		Height: 600,
	}

	err := out.Show()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if mockPresenter.CapturedOutput != out {
		t.Fatalf("expected Show to delegate the correct Output instance")
	}
}

func TestOutput_Show_NoPresenter(t *testing.T) {
	originalPresenter := output.DefaultPresenter
	defer func() { output.DefaultPresenter = originalPresenter }()

	output.DefaultPresenter = nil

	out := &output.Output{}
	err := out.Show()
	if err == nil || err.Error() != "no default presenter configured" {
		t.Fatalf("expected 'no default presenter configured' error, got %v", err)
	}
}

func TestOutput_Export_Delegation(t *testing.T) {
	mockExporter := &MockExporter{}
	originalExporter := output.DefaultExporter
	defer func() { output.DefaultExporter = originalExporter }()

	output.DefaultExporter = mockExporter

	out := &output.Output{
		Width:  400,
		Height: 300,
	}

	testFilename := "test_output.png"
	err := out.Export(testFilename)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if mockExporter.CapturedOutput != out {
		t.Fatalf("expected Export to delegate the correct Output instance")
	}

	if mockExporter.CapturedFile != testFilename {
		t.Fatalf("expected Export to receive the correct filename")
	}
}

func TestOutput_Export_NoExporter(t *testing.T) {
	originalExporter := output.DefaultExporter
	defer func() { output.DefaultExporter = originalExporter }()

	output.DefaultExporter = nil

	out := &output.Output{}
	err := out.Export("dummy.png")
	if err == nil || err.Error() != "no default exporter configured" {
		t.Fatalf("expected 'no default exporter configured' error, got %v", err)
	}
}
