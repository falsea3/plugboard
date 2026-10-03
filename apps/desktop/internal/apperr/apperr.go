package apperr

type Coded interface {
	error
	Code() string
}

type Error struct {
	code string
	text string
}

func New(code, text string) *Error { return &Error{code: code, text: text} }

func (e *Error) Error() string { return e.text }

func (e *Error) Code() string { return e.code }
