package access

import "testing"

func TestAuthenticate(t *testing.T) {
	cases := []struct {
		name     string
		user     string
		password string
		want     Role
		ok       bool
	}{
		{"admin ok", "admin", "1", RoleAdmin, true},
		{"admin padded", " Admin ", "1", RoleAdmin, true},
		{"admin wrong password", "admin", "2", RoleNone, false},
		{"admin empty password", "admin", "", RoleNone, false},
		{"operator empty password", "operator", "", RoleOperator, true},
		{"operator padded", "  OPERATOR ", "", RoleOperator, true},
		{"operator with password", "operator", "x", RoleNone, false},
		{"unknown user", "guest", "", RoleNone, false},
		{"empty user", "", "", RoleNone, false},
	}
	for _, c := range cases {
		got, ok := Authenticate(c.user, c.password)
		if got != c.want || ok != c.ok {
			t.Errorf("%s: Authenticate(%q, %q) = (%v, %v), want (%v, %v)", c.name, c.user, c.password, got, ok, c.want, c.ok)
		}
	}
}

func TestRoleString(t *testing.T) {
	if got := RoleNone.String(); got != "none" {
		t.Errorf("RoleNone.String() = %q, want none", got)
	}
	if got := RoleOperator.String(); got != "operator" {
		t.Errorf("RoleOperator.String() = %q, want operator", got)
	}
	if got := RoleAdmin.String(); got != "admin" {
		t.Errorf("RoleAdmin.String() = %q, want admin", got)
	}
}
