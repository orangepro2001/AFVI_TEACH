package config

import "testing"

// TestExampleConfigLoads keeps configs/afvi.example.toml honest: it must parse and carry the
// window settings the application reads.
func TestExampleConfigLoads(t *testing.T) {
	c, err := Load("../../configs/afvi.example.toml")
	if err != nil {
		t.Fatalf("the shipped example does not load: %v", err)
	}
	if c == nil || len(c.Machines) < 2 {
		t.Fatalf("the example has %d machines", len(c.Machines))
	}
	if c.StartProfile != "" && c.StartProfile != "ask" && c.StartProfile != "teach" && c.StartProfile != "studio" {
		t.Fatalf("start_profile = %q", c.StartProfile)
	}
}
