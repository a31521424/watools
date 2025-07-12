package logger

type Adapter struct {
}

func (a Adapter) Print(msg string) {
	Print(msg)
}

func (a Adapter) Trace(msg string) {
	Trace(msg)
}

func (a Adapter) Debug(msg string) {
	Debug(msg)
}

func (a Adapter) Info(msg string) {
	Info(msg)
}

func (a Adapter) Warning(msg string) {
	Warning(msg)
}

func (a Adapter) Error(msg string) {
	ErrorWithPureString(msg)
}

func (a Adapter) Fatal(msg string) {
	Fatal(msg)
}

func NewAdapter() Adapter {
	return Adapter{}
}
