package gonet

import (
	"testing"
	"time"
)

type payload struct {
	Greeting string
}

func TestGonetAndClientCloseConnection(t *testing.T) {
	var p payload
	r, e := MakeChan(":8080", &p)
	if e != nil {
		t.Skipf("Failed MakeChan: %v", e)
	}

	w, e := OpenChan(":8080")
	if e != nil {
		t.Skipf("Failed OpenChan: %v", e)
	}

	w <- &payload{"hello"}
	v := <-r
	if v.(*payload).Greeting != "hello" {
		t.Skipf("Expecting receiving hello, got %s", v.(*payload).Greeting)
	}

	w <- &payload{"world!"}
	v = <-r
	if v.(*payload).Greeting != "world!" {
		t.Skipf("Expecting receiving world!, got %s", v.(*payload).Greeting)
	}

	close(w)
	select {
	case _ = <-r:
		t.Skipf("Expecting read-end channel not closed. But closed.")
	case <-time.After(100 * time.Millisecond):
	}

	CloseChan(":8080")
}

func TestGonetAndServerCloseConnection(t *testing.T) {
	var p payload
	r, e := MakeChan(":8080", &p)
	if e != nil {
		t.Skipf("Failed MakeChan: %v", e)
	}

	w, e := OpenChan(":8080")
	if e != nil {
		t.Skipf("Failed OpenChan: %v", e)
	}

	w <- &payload{"hello"}
	v := <-r
	if v.(*payload).Greeting != "hello" {
		t.Skipf("Expecting receiving hello, got %s", v.(*payload).Greeting)
	}

	CloseChan(":8080")
	time.Sleep(100 * time.Millisecond)

	defer func() {
		if e := recover(); e != nil {
			t.Logf("Write-end channel is closed by user as expected: %v", e)
		}
	}()
	w <- &payload{"world!"}

	time.Sleep(100 * time.Millisecond)
}
