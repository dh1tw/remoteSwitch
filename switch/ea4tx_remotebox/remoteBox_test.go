package remotebox

import (
	"testing"
)

func TestSerialCommunication(t *testing.T) {

	spPort := Portname("/dev/tty.usbmodem000011")

	rb := New(spPort)

	rb.Name()

	// c := make(chan os.Signal, 1)
	// signal.Notify(c, os.Interrupt)

	// <-c
}
