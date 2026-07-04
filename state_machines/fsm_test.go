package state_machines

import (
	"context"
	"fmt"
	"testing"
)

func TestState(t *testing.T) {
	s := NewShipmentT(456)

	err := s.FSM.Event(context.Background(), "begin")
	if err != nil {
		fmt.Println(err)
	}

	err = s.FSM.Event(context.Background(), "end")
	if err != nil {
		fmt.Println(err)
	}
}
