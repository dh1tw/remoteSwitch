package remotebox

import (
	"os"
	"os/signal"
	"testing"
)

func TestSerialCommunication(t *testing.T) {

	// spPort := Portname("/dev/tty.usbmodem194801")
	spPort := Portname("/dev/tty.usbmodem000011")

	rb, err := New(spPort)
	if err != nil {
		t.Fatal(err)
	}

	rb.Name()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	<-c
}
