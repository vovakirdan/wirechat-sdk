package wirechat

// Dispatcher routes outbound events to registered callbacks.
type Dispatcher struct {
	onMessage    func(MessageEvent)
	onUserJoined func(UserEvent)
	onUserLeft   func(UserEvent)
	onHistory    func(HistoryEvent)
	onError      func(error)
}

func (d *Dispatcher) SetOnMessage(fn func(MessageEvent)) { d.onMessage = fn }
func (d *Dispatcher) SetOnUserJoined(fn func(UserEvent)) { d.onUserJoined = fn }
func (d *Dispatcher) SetOnUserLeft(fn func(UserEvent))   { d.onUserLeft = fn }
func (d *Dispatcher) SetOnHistory(fn func(HistoryEvent)) { d.onHistory = fn }
func (d *Dispatcher) SetOnError(fn func(error))          { d.onError = fn }

func (d *Dispatcher) Dispatch(out Outbound) {
	if out.Type == outboundError && out.Error != nil && d.onError != nil {
		// Convert protocol error to WirechatError
		d.onError(FromProtocolError(out.Error))
		return
	}
	switch out.Event {
	case eventMessage:
		if d.onMessage == nil {
			return
		}
		var ev MessageEvent
		if err := UnmarshalData(out.Data, &ev); err != nil {
			d.fireError(WrapError(ErrorSerialization, "failed to unmarshal message event", err))
			return
		}
		d.onMessage(ev)
	case eventUserJoined:
		if d.onUserJoined == nil {
			return
		}
		var ev UserEvent
		if err := UnmarshalData(out.Data, &ev); err != nil {
			d.fireError(WrapError(ErrorSerialization, "failed to unmarshal user_joined event", err))
			return
		}
		d.onUserJoined(ev)
	case eventUserLeft:
		if d.onUserLeft == nil {
			return
		}
		var ev UserEvent
		if err := UnmarshalData(out.Data, &ev); err != nil {
			d.fireError(WrapError(ErrorSerialization, "failed to unmarshal user_left event", err))
			return
		}
		d.onUserLeft(ev)
	case eventHistory:
		if d.onHistory == nil {
			return
		}
		var ev HistoryEvent
		if err := UnmarshalData(out.Data, &ev); err != nil {
			d.fireError(WrapError(ErrorSerialization, "failed to unmarshal history event", err))
			return
		}
		d.onHistory(ev)
	}
}

func (d *Dispatcher) fireError(err error) {
	if d.onError != nil && err != nil {
		d.onError(err)
	}
}
