package base

type Error struct {
	Code    int
	Message string
}

func (e Error) Error() string {
	return e.Message
}

func (e *Error) ErrorF(msg string) {
	e.Message = e.Message + ": " + msg
}
