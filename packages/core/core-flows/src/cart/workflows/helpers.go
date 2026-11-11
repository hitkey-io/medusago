package workflows

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
)

func equalBN(a, b any) bool { return totals.Eq(a, b) }

type plainErr struct{ msg string }

func (p plainErr) Error() string { return p.msg }

// plainError mirrors throw new Error(message) (an unknown error, 500).
func plainError(msg string) error { return plainErr{msg} }
