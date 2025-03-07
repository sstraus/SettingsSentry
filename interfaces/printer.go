package interfaces

// Printer defines an interface for printing messages
type Printer interface {
	// Print formats and prints a message
	Print(format string, args ...interface{})
	// Reset resets the printer state
	Reset()
}
