package binarygen

//go:generate binary-gen -t User -o types_gen.go
type User struct {
	ID           uint64
	RegisteredAt uint64
	Year         uint16
	Status       uint8
	ExtID        uint32
}
