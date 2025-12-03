package wirechat

import (
	"context"
	"encoding/json"
	"testing"
)

func TestDispatcherMessage(t *testing.T) {
	var got MessageEvent
	var errCalled bool
	var d Dispatcher
	d.SetOnMessage(func(ev MessageEvent) { got = ev })
	d.SetOnError(func(err error) { errCalled = true; _ = err })

	raw, _ := json.Marshal(MessageEvent{Room: "general", User: "alice", Text: "hi"})
	d.Dispatch(Outbound{Type: outboundEvent, Event: eventMessage, Data: raw})

	if got.Room != "general" || got.User != "alice" || got.Text != "hi" {
		t.Fatalf("unexpected event: %+v", got)
	}
	if errCalled {
		t.Fatalf("unexpected error callback")
	}
}

func TestDispatcherError(t *testing.T) {
	var errGot error
	var d Dispatcher
	d.SetOnError(func(err error) { errGot = err })

	d.Dispatch(Outbound{Type: outboundError, Error: &Error{Code: "unauthorized", Msg: "no token"}})
	if errGot == nil {
		t.Fatalf("expected error callback")
	}
}

func TestClientSendNotConnected(t *testing.T) {
	cfg := DefaultConfig()
	c := NewClient(&cfg)
	err := c.Send(testCtx(), "room", "hi")
	if err == nil {
		t.Fatalf("expected error when not connected")
	}
}

// testCtx returns a cancellable context for unit tests.
func testCtx() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}
