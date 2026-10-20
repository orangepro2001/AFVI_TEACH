// Package access holds the login check of the two programs: who may use the full
// engineering interface and who only sees the guided operator pages. It is pure Go
// (standard library only) so it builds and tests without cgo.
package access

import (
	"crypto/sha256"
	"crypto/subtle"
	"strings"
)

// Role is the access level of the person at the keyboard.
type Role int

const (
	RoleNone     Role = iota // nobody logged in (the studio before its login)
	RoleOperator             // ordinary user: guided pages only
	RoleAdmin                // the full engineering interface (what the app had before)
)

// String returns the stable name of the role ("none" | "operator" | "admin").
func (r Role) String() string {
	switch r {
	case RoleOperator:
		return "operator"
	case RoleAdmin:
		return "admin"
	}
	return "none"
}

// The clear-text passwords are never stored; only their SHA-256 digests are.
var (
	adminPasswordDigest = sha256.Sum256([]byte("1"))
	emptyPasswordDigest = sha256.Sum256(nil)
)

// Authenticate checks a login. user is matched case-insensitively after trimming spaces.
// "admin" with password "1" -> RoleAdmin. "operator" with an empty password -> RoleOperator.
// Anything else -> (RoleNone, false). The password is compared as SHA-256 digests with
// crypto/subtle.ConstantTimeCompare.
func Authenticate(user, password string) (Role, bool) {
	name := strings.ToLower(strings.TrimSpace(user))
	digest := sha256.Sum256([]byte(password))
	switch name {
	case "admin":
		if subtle.ConstantTimeCompare(digest[:], adminPasswordDigest[:]) == 1 {
			return RoleAdmin, true
		}
	case "operator":
		if subtle.ConstantTimeCompare(digest[:], emptyPasswordDigest[:]) == 1 {
			return RoleOperator, true
		}
	}
	return RoleNone, false
}
