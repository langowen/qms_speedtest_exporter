package entities

import "errors"

// Общие ошибки домена
var (
	ErrTimeout       = errors.New("timeout")
	ErrCancelled     = errors.New("cancelled")
	ErrExecFailed    = errors.New("external_exec_failed")
	ErrResultMissing = errors.New("result_file_missing")
)

// ErrorResponse — стандарт для ответа об ошибке между сервисами.
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
