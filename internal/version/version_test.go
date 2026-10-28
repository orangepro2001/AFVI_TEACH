package version

import "testing"

func TestUnstampedBuildKnowsNothing(t *testing.T) {
	// A build that cannot name itself must not take part in updates: "dev" is what go run and
	// go test produce, and the update check stays off for it.
	if Name != "dev" || CodeText != "" || Code() != 0 {
		t.Fatalf("defaults are %q/%q, want dev/empty", Name, CodeText)
	}
	if Known() {
		t.Fatal("an unstamped build reports Known()")
	}
	if got := String(); got != "dev" {
		t.Fatalf("String() = %q, want dev", got)
	}
}

func TestStampedBuild(t *testing.T) {
	// the values are set through -ldflags, so the test writes them directly
	oldName, oldCode := Name, CodeText
	defer func() { Name, CodeText = oldName, oldCode }()
	Name, CodeText = "0.1.12", "112"
	if !Known() {
		t.Fatal("a stamped build reports !Known()")
	}
	if got := String(); got != "0.1.12 (112)" {
		t.Fatalf("String() = %q, want 0.1.12 (112)", got)
	}
	// a name without a code (or the other way round) is still unstamped
	CodeText = ""
	if Known() {
		t.Fatal("a build with code 0 reports Known()")
	}
}
