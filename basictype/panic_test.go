package basictype

import (
	"fmt"
	"testing"
)

func TestDefer(t *testing.T) {
	err := example()
	if err == nil {
		t.Error("expected panic but got nil")
	}
	fmt.Println("Test recovered from panic:", err)
}

func example() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic recovered: %v", r)
		}
	}()

	defer fmt.Println("world")

	fmt.Println("before panic hello")

	panic("here")
}
