package BandswitchGPIO

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	sw "github.com/dh1tw/remoteSwitch/switch"
	"periph.io/x/periph/conn/gpio"
	"periph.io/x/periph/conn/gpio/gpioreg"
	"periph.io/x/periph/host"
)

type BandswitchGPIO struct {
	sync.Mutex
	name         string
	id           int
	ports        map[string]*port
	portConfig   map[string]PortConfig
	eventHandler func(sw.Switcher, sw.Device)
}

type port struct {
	name         string
	activeRelays map[string]*relay
	relays       map[string]*relay
	id           int
}

type relay struct {
	name     string
	inverted bool
	state    bool
	pin      gpio.PinOut
	id       int
}

func (r *relay) set(state bool) error {

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

func (r *relay) getState() bool {
	if r.inverted {
		return !r.state
	}
	return r.state
}

func NewSwitchGPIO(options ...func(*BandswitchGPIO)) *BandswitchGPIO {

	g := &BandswitchGPIO{
		name:       "My Bandswitch",
		id:         0,
		ports:      make(map[string]*port),
		portConfig: make(map[string]PortConfig),
	}

	for _, opt := range options {
		opt(g)
	}

	return g
}

func (g *BandswitchGPIO) Init() error {

	if _, err := host.Init(); err != nil {
		return err
	}

	for pName, pConfig := range g.portConfig {
		if _, portNameExists := g.ports[pName]; portNameExists {
			return fmt.Errorf("portname %s already exists", pName)
		}
		p := &port{
			name:         pConfig.Name,
			relays:       make(map[string]*relay),
			activeRelays: make(map[string]*relay),
			id:           pConfig.ID,
		}

		for _, pinConfig := range pConfig.OutPorts {
			r := &relay{
				name:     pinConfig.Name,
				inverted: pinConfig.Inverted,
				id:       pinConfig.ID,
			}

			//TBD Handle pin "None" / Empty to disable all relays
			pin := gpioreg.ByName(strings.ToUpper(pinConfig.Pin))

			if pin == nil {
				return fmt.Errorf("failed to find pin %s", pinConfig.Pin)
			}

			r.pin = pin
			if err := r.set(false); err != nil {
				return err
			}
			p.relays[pinConfig.Name] = r
		}

		g.ports[pName] = p
	}
	return nil
}

func (g *BandswitchGPIO) Name() string {
	return g.name
}

func (g *BandswitchGPIO) SetPort(portRequest sw.Port) error {
	g.Lock()
	defer g.Unlock()

	// ensure that the requested port exists
	p, ok := g.ports[portRequest.Name]
	if !ok {
		return fmt.Errorf("%s is an invalid input port", portRequest.Name)
	}

	// ensure that the requested terminal exists
	for _, t := range portRequest.Terminals {
		_, ok := p.relays[t.Name]
		if !ok {
			return fmt.Errorf("%s is an invalid terminal", t.Name)
		}
	}

	//ensure that the requested terminal is not in use by any other port
	for prtName, prt := range g.ports {

		// only check the remaining ports
		if prtName == portRequest.Name {
			continue
		}

		for _, t := range portRequest.Terminals {
			if _, found := prt.activeRelays[t.Name]; found {
				return fmt.Errorf("terminal %s in use by port %s", t.Name, prtName)
			}
		}
	}

	// deactivate all relays on this port
	for rName, r := range p.activeRelays {

		if err := r.set(false); err != nil {
			return err
		}
		// remove from the map of active relays
		delete(p.activeRelays, rName)
	}

	// activate relay
	for _, t := range portRequest.Terminals {
		r := p.relays[t.Name]
		if err := r.set(true); err != nil {
			return err
		}
		// add to the map of active relays
		p.activeRelays[t.Name] = r
	}

	if g.eventHandler != nil {
		device := g.serialize()
		go g.eventHandler(g, device)
	}

	return nil
}

func (g *BandswitchGPIO) GetPort(portName string) (sw.Port, error) {
	g.Lock()
	defer g.Unlock()

	p, ok := g.ports[portName]
	if !ok {
		return sw.Port{}, fmt.Errorf("%s in an invalid port", portName)
	}

	return p.serialize(), nil
}

func (g *BandswitchGPIO) Serialize() sw.Device {
	g.Lock()
	defer g.Unlock()

	return g.serialize()
}

func (p *port) serialize() sw.Port {
	swPort := sw.Port{
		Name:      p.name,
		ID:        p.id,
		Terminals: []sw.Terminal{},
	}

	for _, r := range p.relays {
		t := sw.Terminal{
			Name:  r.name,
			ID:    r.id,
			State: r.getState(),
		}
		swPort.Terminals = append(swPort.Terminals, t)
	}

	// sort the Terminals by id
	sort.Slice(swPort.Terminals, func(i, j int) bool {
		return swPort.Terminals[i].ID < swPort.Terminals[j].ID
	})

	return swPort
}

func (g *BandswitchGPIO) serialize() sw.Device {

	dev := sw.Device{
		Name: g.name,
	}

	// serialize all ports
	for _, p := range g.ports {
		swPort := p.serialize()
		dev.Ports = append(dev.Ports, swPort)
	}

	// sort the ports by ID
	sort.Slice(dev.Ports, func(i, j int) bool {
		return dev.Ports[i].ID < dev.Ports[j].ID
	})

	return dev
}

func (g *BandswitchGPIO) Close() {
	for _, p := range g.ports {
		for _, r := range p.relays {
			r.set(false)
		}
	}
}
