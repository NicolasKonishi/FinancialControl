package repository

import "errors"

var (
	ErrNotFound      = errors.New("not found")
	ErrNoWallet      = errors.New("member has no wallet")
	ErrWalletOwner   = errors.New("wallet does not belong to member")
	ErrInsufficient  = errors.New("insufficient savings")
	ErrNotCredit     = errors.New("wallet is not a credit card")
	ErrInvalidAmount = errors.New("invalid amount")
	ErrInvoiceEmpty  = errors.New("invoice is already paid")
)
