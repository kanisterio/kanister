package log

type Printer interface {
	Print(msg string, fields ...F)
}
