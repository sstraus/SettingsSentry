package interfaces

type Printer interface {
	Print(format string, args ...interface{})
	Reset()
}
