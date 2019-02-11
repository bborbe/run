package run

import (
	"fmt"
	"testing"
)

func TestNewEmptyError(t *testing.T) {
	err := NewErrorList()
	if err != nil {
		t.Fatalf("nil expected")
	}
}

func TestNewErorrList(t *testing.T) {
	err := NewErrorList(fmt.Errorf("test"))
	if err == nil {
		t.Fatalf("nil not expected")
	}
	if err.Error() != "errors: test" {
		t.Fatalf("invalid msg")
	}
}

func TestNewByChanEmptyError(t *testing.T) {
	c := make(chan error, 10)
	close(c)
	err := NewErrorListByChan(c)
	if err != nil {
		t.Fatalf("nil expected")
	}
}

func TestNewByChanErorrList(t *testing.T) {
	c := make(chan error, 10)
	c <- fmt.Errorf("test")
	close(c)
	err := NewErrorListByChan(c)
	if err == nil {
		t.Fatalf("nil not expected")
	}
	if err.Error() != "errors: test" {
		t.Fatalf("invalid msg")
	}
}
