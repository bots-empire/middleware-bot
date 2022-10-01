package model

const (
	// ErrCommandNotConverted error command not recognize.
	ErrCommandNotConverted = Error("command not converted")
	// ErrUserNotFound error user not found.
	ErrUserNotFound = Error("user not found")
	// ErrFoundTwoUsers error found two user account for one user.
	ErrFoundTwoUsers = Error("found two users")
	// ErrNotAdminUser error not admin user.
	ErrNotAdminUser = Error("not admin user")
	// ErrMoreMoneyButtonUnavailable error more money button unavailable.
	ErrMoreMoneyButtonUnavailable = Error("more money button unavailable")

	// ErrScanSqlRow error scan sql row.
	ErrScanSqlRow = Error("failed scan sql row")
	// ErrNotEnoughParameters error not enough parameters.
	ErrNotEnoughParameters = Error("not enough parameters")

	// ErrRedisNil error redis: nil.
	ErrRedisNil = Error("redis: nil")
)

type Error string

func (e Error) Error() string {
	return string(e)
}
