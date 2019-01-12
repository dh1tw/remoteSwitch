package DummySwitch

import (
	"fmt"
	"sort"
	"sync"

	sw "github.com/dh1tw/remoteSwitch/switch"
)

type DummySwitch struct {
	sync.RWMutex
	name         string
	index        int
	exclusive    bool
	ports        map[string]*port
	switchConfig SwitchConfig
	eventHandler func(sw.Switcher, sw.Device)
}

type port struct {
	name            string
	activeTerminals map[string]*terminal
	terminals       map[string]*terminal
	exclusive       bool
	index           int
}

type terminal struct {
	name     string
	inverted bool
	state    bool
	index    int
}

func NewDummySwitch(options ...func(*DummySwitch)) *DummySwitch {

	d := &DummySwitch{
		name:  "my DummySwitch",
		index: 0,
		ports: make(map[string]*port),
	}

	for _, opt := range options {
		opt(d)
	}

	return d
}

func (d *DummySwitch) Init() error {
	d.name = d.switchConfig.Name
	d.index = d.switchConfig.Index
	d.exclusive = d.switchConfig.Exclusive

	for _, pConfig := range d.switchConfig.Ports {
		if _, portNameExists := d.ports[pConfig.Name]; portNameExists {
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
				name:  pinConfig.Name,
				index: pinConfig.Index,
			}

			p.terminals[pinConfig.Name] = r
		}

		d.ports[pConfig.Name] = p
	}
	return nil
}

// Name returns the Name of this Dummy Switch
func (d *DummySwitch) Name() string {
	d.RLock()
	defer d.RUnlock()
	return d.name
}

// SetPort sets the Terminals of a particular Port. The portRequest
// can contain n termials.
func (d *DummySwitch) SetPort(portRequest sw.Port) error {
	d.Lock()
	defer d.Unlock()

	// ensure that the requested port exists
	p, ok := d.ports[portRequest.Name]
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

	// if DummySwitch.exclusive is true, a particular terminal can only
	// be active on one port
	if d.exclusive {
		//ensure that the requested terminal is not in use by any other port
		for prtName, prt := range d.ports {

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

			r.state = false

			// remove from the map of active relays
			delete(p.activeTerminals, rName)
		}
	}

	// set state of the terminal
	for _, t := range portRequest.Terminals {
		r := p.terminals[t.Name]

		r.state = t.State

		if t.State {
			// add to the map of active terminals
			p.activeTerminals[t.Name] = r
			continue
		}

		// when false, remove from map of active terminals
		delete(p.activeTerminals, t.Name)

	}

	if d.eventHandler != nil {
		device := d.serialize()
		go d.eventHandler(d, device)
	}

	return nil
}

// GetPort returns switch.Port struct containing the current state of
// the requested port.
func (d *DummySwitch) GetPort(portName string) (sw.Port, error) {
	d.RLock()
	defer d.RUnlock()

	p, ok := d.ports[portName]
	if !ok {
		return sw.Port{}, fmt.Errorf("%s in an invalid port", portName)
	}

	return p.serialize(), nil
}

// Serialize returns a switch.Device struct containing the current
// state and configuration of this Dummy switch.
func (d *DummySwitch) Serialize() sw.Device {
	d.RLock()
	defer d.RUnlock()

	return d.serialize()
}

// serialize returns a switch.Port struct containing the current
// state and configuration of this Port. This method
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
			State: r.state,
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
// state and configuration of this Dummy switch. This method
// is not threadsafe.
func (d *DummySwitch) serialize() sw.Device {

	dev := sw.Device{
		Name:  d.name,
		Index: d.index,
	}

	// serialize all ports
	for _, p := range d.ports {
		swPort := p.serialize()
		dev.Ports = append(dev.Ports, swPort)
	}

	// sort the ports by index
	sort.Slice(dev.Ports, func(i, j int) bool {
		return dev.Ports[i].Index < dev.Ports[j].Index
	})

	return dev
}

// Close shuts down the switch.
func (d *DummySwitch) Close() {
	// nothing to do
}
