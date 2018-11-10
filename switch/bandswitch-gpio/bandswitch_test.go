package BandswitchGPIO

import (
	"testing"
	"time"
)

var configA = PortConfig{
	Name: "A",
	OutPorts: []PinConfig{
		PinConfig{
			Name:     "Port1A",
			Pin:      "GPIO3",
			Inverted: true,
		},
		PinConfig{
			Name:     "Port2A",
			Pin:      "GPIO19",
			Inverted: true,
		},
		PinConfig{
			Name:     "Port3A",
			Pin:      "GPIO18",
			Inverted: true,
		},
		PinConfig{
			Name:     "Port4A",
			Pin:      "GPIO15",
			Inverted: true,
		},
		PinConfig{
			Name:     "Port5A",
			Pin:      "GPIO16",
			Inverted: true,
		},
		PinConfig{
			Name:     "Port6A",
			Pin:      "GPIO2",
			Inverted: true,
		},
		PinConfig{
			Name:     "Port7A",
			Pin:      "GPIO14",
			Inverted: true,
		},
		PinConfig{
			Name:     "Port8A",
			Pin:      "GPIO13",
			Inverted: true,
		},
	},
}

var configB = PortConfig{
	Name: "B",
	OutPorts: []PinConfig{
		PinConfig{
			Name:     "Port1B",
			Pin:      "GPIO7",
			Inverted: true,
		},
		PinConfig{
			Name:     "Port2B",
			Pin:      "GPIO0",
			Inverted: true,
		},
		PinConfig{
			Name:     "Port3B",
			Pin:      "GPIO199",
			Inverted: true,
		},
		PinConfig{
			Name:     "Port4B",
			Pin:      "GPIO1",
			Inverted: true,
		},
		PinConfig{
			Name:     "Port5B",
			Pin:      "GPIO6",
			Inverted: true,
		},
		PinConfig{
			Name:     "Port6B",
			Pin:      "GPIO198",
			Inverted: true,
		},
		PinConfig{
			Name:     "Port7B",
			Pin:      "GPIO12",
			Inverted: true,
		},
		PinConfig{
			Name:     "Port8B",
			Pin:      "GPIO11",
			Inverted: true,
		},
	},
}

func TestGPIOInitialization(t *testing.T) {

	rfSwitch := NewGpioSwitch(Port(configA), Port(configB))
	if err := rfSwitch.Init(); err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Second)

	a, _ := rfSwitch.GetPort("A")
	b, _ := rfSwitch.GetPort("B")
	t.Logf("a: %s, b: %s", a, b)

	if err := rfSwitch.SetPort("A", "20m"); err != nil {
		t.Fatal(err)
	}
	a, _ = rfSwitch.GetPort("A")
	b, _ = rfSwitch.GetPort("B")
	t.Logf("a: %s, b: %s", a, b)
	time.Sleep(time.Millisecond * 500)

	if err := rfSwitch.SetPort("A", "40m"); err != nil {
		t.Fatal(err)
	}
	a, _ = rfSwitch.GetPort("A")
	b, _ = rfSwitch.GetPort("B")
	t.Logf("a: %s, b: %s", a, b)

	if err := rfSwitch.SetPort("B", "20m"); err != nil {
		t.Fatal(err)
	}
	a, _ = rfSwitch.GetPort("A")
	b, _ = rfSwitch.GetPort("B")
	t.Logf("a: %s, b: %s", a, b)

	time.Sleep(time.Millisecond * 500)
	if err := rfSwitch.SetPort("A", "20m"); err != nil {
		t.Fatal(err)
	}
	a, _ = rfSwitch.GetPort("A")
	b, _ = rfSwitch.GetPort("B")
	t.Logf("a: %s, b: %s", a, b)

	time.Sleep(time.Millisecond * 500)

	rfSwitch.Close()
}
