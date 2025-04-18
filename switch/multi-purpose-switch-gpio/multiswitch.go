package MultiPurposeSwitchGPIO

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	sw "github.com/dh1tw/remoteSwitch/switch"
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/host/v3"
)

// MPSwitchGPIO contains the state and configuration of a
// multi purpose GPIO switch
type MPSwitchGPIO struct {
	sync.Mutex
	name         string
	index        int
	exclusive    bool
	ports        map[string]*port
	switchConfig SwitchConfig
	eventHandler func(sw.Switcher, sw.Device)
}

// port represents a set of terminals (GPIO pins). This struct holds the
// configuration and state of the port.
type port struct {
	name            string
	activeTerminals map[string]*terminal
	terminals       map[string]*terminal
	exclusive       bool
	index           int
}

// terminal represents a particular GPIO pin. This struct holds the
// configuration and state of the terminal.
type terminal struct {
	name     string
	inverted bool
	state    bool
	pin      gpio.PinOut
	index    int
}

// NewMPSwitchGPIO is the constructor for a Multi Purpose GPIO switch.
// The constructor takes functional arguments for configuring the MPSwitch.
func NewMPSwitchGPIO(options ...func(*MPSwitchGPIO)) *MPSwitchGPIO {

	g := &MPSwitchGPIO{
		name:  "My Bandswitch",
		index: 0,
		ports: make(map[string]*port),
	}

	for _, opt := range options {
		opt(g)
	}

	return g
}

// Init intializes the Multi Purpose GPIO   Switch.
// If your platform does not support GPIO, an error will be returned.
func (g *MPSwitchGPIO) Init() error {

	hostState, err := host.Init()
	if err != nil {
		return err
	}

	// check if sysfs-gpio driver has been loaded
	gpioDriverLoaded := false
	for _, driver := range hostState.Loaded {
		if driver.String() == "sysfs-gpio" {
			gpioDriverLoaded = true
		}
	}

	if !gpioDriverLoaded {
		return fmt.Errorf("sysfs-gpio driver was not loaded; try running as root")
	}

	g.name = g.switchConfig.Name
	g.index = g.switchConfig.Index
	g.exclusive = g.switchConfig.Exclusive

	for _, pConfig := range g.switchConfig.Ports {
		if _, portNameExists := g.ports[pConfig.Name]; portNameExists {
			return fmt.Errorf("portname %s already exists", pConfig.Name)
		}
		p := &port{
			name:            pConfig.Name,
			terminals:       make(map[string]*terminal),
			activeTerminals: make(map[string]*terminal),
			exclusive:       pConfig.Exclusive,
			index:           pConfig.Index,
		}

		for _, pinConfig := range pConfig.Terminals {
			r := &terminal{
				name:     pinConfig.Name,
				inverted: pinConfig.Inverted,
				index:    pinConfig.Index,
				pin:      gpioreg.ByName(strings.ToUpper(pinConfig.Pin)),
			}

			//TBD Handle pin "None" / Empty to disable all relays

			if r.pin == nil {
				return fmt.Errorf("failed to find pin %s", pinConfig.Pin)
			}

			if err := r.setState(false); err != nil {
				return err
			}
			p.terminals[pinConfig.Name] = r
		}

		g.ports[pConfig.Name] = p
	}
	return nil
}

// Name returns the Name of this Multi Purpose GPIO Switch
func (g *MPSwitchGPIO) Name() string {
	g.Lock()
	defer g.Unlock()
	return g.name
}

// SetPort sets the Terminals of a particular Port. The portRequest
// can contain n termials.
func (g *MPSwitchGPIO) SetPort(portRequest sw.Port) error {
	g.Lock()
	defer g.Unlock()

	// ensure that the requested port exists
	p, ok := g.ports[portRequest.Name]
	if !ok {
		return fmt.Errorf("%s is an invalid port", portRequest.Name)
	}

	// ensure that the requested terminal exists
	for _, t := range portRequest.Terminals {
		_, ok := p.terminals[t.Name]
		if !ok {
			return fmt.Errorf("%s is an invalid terminal", t.Name)
		}
	}

	// if MPSwitchGPIO.exclusive is true, a particular terminal can only
	// be active on one port
	if g.exclusive {
		//ensure that the requested terminal is not in use by any other port
		for prtName, prt := range g.ports {

			// only check the remaining ports
			if prtName == portRequest.Name {
				continue
			}

			for _, t := range portRequest.Terminals {
				if _, found := prt.activeTerminals[t.Name]; found {
					return fmt.Errorf("terminal %s in use by port %s",
						t.Name, prtName)
				}
			}
		}
	}

	// if port.exclusive is enabled, only one terminal can be active
	// on this port.
	if p.exclusive {
		// deactivate all relays on this port
		for rName, r := range p.activeTerminals {

			if err := r.setState(false); err != nil {
				return err
			}
			// remove from the map of active relays
			delete(p.activeTerminals, rName)
		}
	}

	// set state of the terminal
	for _, t := range portRequest.Terminals {
		r := p.terminals[t.Name]

		if err := r.setState(t.State); err != nil {
			return err
		}

		if t.State {
			// add to the map of active terminals
			p.activeTerminals[t.Name] = r
			continue
		}

		// when false, remove from map of active terminals
		delete(p.activeTerminals, t.Name)

	}

	if g.eventHandler != nil {
		device := g.serialize()
		go g.eventHandler(g, device)
	}

	return nil
}

// GetPort returns switch.Port struct containing the current state of
// the requested port.
func (g *MPSwitchGPIO) GetPort(portName string) (sw.Port, error) {
	g.Lock()
	defer g.Unlock()

	p, ok := g.ports[portName]
	if !ok {
		return sw.Port{}, fmt.Errorf("%s in an invalid port", portName)
	}

	return p.serialize(), nil
}

// Serialize returns a switch.Device struct containing the current
// state and configuration of this MultiPurpose GPIO switch.
func (g *MPSwitchGPIO) Serialize() sw.Device {
	g.Lock()
	defer g.Unlock()

	return g.serialize()
}

// serialize returns a switch.Port struct containing the current
// state and configuration of this GPIO Port. This method
// is not threadsafe.
func (p *port) serialize() sw.Port {
	swPort := sw.Port{
		Name:      p.name,
		Index:     p.index,
		Terminals: []sw.Terminal{},
	}

	for _, r := range p.terminals {
		t := sw.Terminal{
			Name:  r.name,
			Index: r.index,
			State: r.getState(),
		}
		swPort.Terminals = append(swPort.Terminals, t)
	}

	// sort the Terminals by index
	sort.Slice(swPort.Terminals, func(i, j int) bool {
		return swPort.Terminals[i].Index < swPort.Terminals[j].Index
	})

	return swPort
}

// serialize returns a switch.Device struct containing the current
// state and configuration of this MultiPurpose GPIO switch. This method
// is not threadsafe.
func (g *MPSwitchGPIO) serialize() sw.Device {

	dev := sw.Device{
		Name:  g.name,
		Index: g.index,
	}

	// serialize all ports
	for _, p := range g.ports {
		swPort := p.serialize()
		dev.Ports = append(dev.Ports, swPort)
	}

	// sort the ports by index
	sort.Slice(dev.Ports, func(i, j int) bool {
		return dev.Ports[i].Index < dev.Ports[j].Index
	})

	return dev
}

// setState is a convenience function for setting the relay. It is necessary in
// case the logic is inverted.
func (r *terminal) setState(state bool) error {

	newState := state
	if r.inverted {
		newState = !newState
	}

	if err := r.pin.Out(gpio.Level(newState)); err != nil {
		return err
	}

	r.state = newState

	return nil
}

// getState returns the current state of a terminal / GPIO pin.
func (r *terminal) getState() bool {
	if r.inverted {
		return !r.state
	}
	return r.state
}

// Close shutsdown the switch and sets all GPIO ports to false.
func (g *MPSwitchGPIO) Close() {
	g.Lock()
	defer g.Unlock()

	for _, p := range g.ports {
		for _, r := range p.terminals {
			r.setState(false)
		}
	}
}
