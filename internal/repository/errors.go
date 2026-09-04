package repository

import "errors"

var (
	ErrNotFound              = errors.New("not found")
	ErrNoWallet              = errors.New("member has no wallet")
	ErrWalletOwner           = errors.New("wallet does not belong to member")
	ErrInsufficient          = errors.New("insufficient savings")
	ErrNotCredit             = errors.New("wallet is not a credit card")
	ErrInvalidAmount         = errors.New("invalid amount")
	ErrInvoiceEmpty          = errors.New("invoice is already paid")
	ErrStatementImported     = errors.New("statement already imported")
	ErrStatementTypeMismatch = errors.New("statement type does not match import mode")
)

// DuplicateStatementError is a user-facing refusal of a repeated statement file or month.
type DuplicateStatementError struct {
	Message string
}

func (e DuplicateStatementError) Error() string { return e.Message }

func (e DuplicateStatementError) Unwrap() error { return ErrStatementImported }
