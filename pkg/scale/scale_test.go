package scale

import (
	"math"
	"testing"
)

func TestContinuous_Map(t *testing.T) {
	c := &Continuous{Min: 10, Max: 20, hasData: true}

	if v := c.Map(15); v != 0.5 {
		t.Errorf("expected 0.5, got %f", v)
	}

	if v := c.Map(10); v != 0.0 {
		t.Errorf("expected 0.0, got %f", v)
	}

	if v := c.Map(20); v != 1.0 {
		t.Errorf("expected 1.0, got %f", v)
	}

	if v := c.Map(25); v != 1.5 {
		t.Errorf("expected 1.5 (extrapolated), got %f", v)
	}
}

func TestContinuous_Reversed(t *testing.T) {
	c := &Continuous{Min: 10, Max: 20, hasData: true, Reversed: true}

	if v := c.Map(15); v != 0.5 {
		t.Errorf("expected 0.5, got %f", v)
	}

	if v := c.Map(10); v != 1.0 {
		t.Errorf("expected 1.0, got %f", v)
	}

	if v := c.Map(20); v != 0.0 {
		t.Errorf("expected 0.0, got %f", v)
	}
}

func TestContinuous_Missing(t *testing.T) {
	c := &Continuous{hasData: false}
	if !math.IsNaN(c.Map(15)) {
		t.Errorf("expected NaN without data")
	}

	c = &Continuous{Min: 10, Max: 20, hasData: true}
	if !math.IsNaN(c.Map(math.NaN())) {
		t.Errorf("expected NaN for NaN input")
	}
}

func TestDiscrete_Map(t *testing.T) {
	d := &Discrete{
		Domain: []string{"A", "B", "C"},
		set: map[string]int{
			"A": 0,
			"B": 1,
			"C": 2,
		},
	}

	if v := d.Map("A"); v != 0.0 {
		t.Errorf("expected 0.0, got %f", v)
	}

	if v := d.Map("B"); v != 0.5 {
		t.Errorf("expected 0.5, got %f", v)
	}

	if v := d.Map("C"); v != 1.0 {
		t.Errorf("expected 1.0, got %f", v)
	}

	if !math.IsNaN(d.Map("Missing")) {
		t.Errorf("expected NaN for unmapped string")
	}
}
