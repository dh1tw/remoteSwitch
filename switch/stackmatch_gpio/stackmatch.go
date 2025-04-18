package StackmatchGPIO

// StackmatchGPIO implements a switchable antenna combiner where the relays
// are driven by GPIO pins.
// A Stackmatch has a set of combinations, which are composed of Terminals
// (the actual selected antenna terminals) and the corresponding relays.

import (
	"fmt"
	"sort"
	"sync"

	sw "github.com/dh1tw/remoteSwitch/switch"
	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
	"periph.io/x/host/v3"
)

// SmGPIO contains the state and configuration of a gpio based stackmatch
type SmGPIO struct {
	sync.RWMutex
	name         string
	portName     string
	index        int
	combinations map[string]*combination
	terminals    []*terminal
	pins         []*pin
	config       SmConfig
	eventHandler func(sw.Switcher, sw.Device)
}

// combination holds for a given amount the terminals, the corresponding
// relay (GPIO pin) configuration.
type combination struct {
	name      string
	terminals map[string]*terminal
	relays    []*pin
}

// terminal describes a particular terminal of the stackmatch. The terminal
// name is typically shown in the GUI as a selectable item.
type terminal struct {
	name  string
	index int
	state bool
}

// pin holds the information associated to the pin / gpio pin.
type pin struct {
	inverted bool
	state    bool
	pin      gpio.PinOut
}

// NewStackmatchGPIO is the constructor for a GPIO based stackmatch (antenna combiner).
// The constructor takes functional arguments for configuring the SmGPIO.
func NewStackmatchGPIO(options ...func(*SmGPIO)) *SmGPIO {

	s := &SmGPIO{
		name:         "myStackmatch",
		index:        100,
		portName:     "SM",
		combinations: make(map[string]*combination),
		terminals:    []*terminal{},
		pins:         []*pin{},
	}

	for _, opt := range options {
		opt(s)
	}

	return s
}

func (s *SmGPIO) Init() error {
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

	s.name = s.config.Name
	s.index = s.config.Index

	// in these maps we will store temporarily the terminals and pins
	// maps are used just for de-duplication
	terminals := make(map[string]*terminal)
	pins := make(map[string]*pin)

	for _, cConfig := range s.config.Combinations {

		newCombination := &combination{
			relays:    []*pin{},
			terminals: make(map[string]*terminal),
		}

		// create the pins for this combination
		for _, pc := range cConfig.Pins {

			newPin, ok := pins[pc.Name]
			// only create the pin if it doesn't exist yet
			if !ok {
				newPin = &pin{
					inverted: pc.Inverted,
					pin:      gpioreg.ByName(pc.Pin),
				}
				if newPin.pin == nil {
					return fmt.Errorf("failed to find pin %s", pc.Name)
				}
				pins[pc.Name] = newPin

			}

			newCombination.relays = append(newCombination.relays, newPin)
		}

		// temporary storage for the combination name (concat of all terminal names)
		tNames := []string{}

		// create the terminals for this combination
		for _, tc := range cConfig.Terminals {

			newTerminal, ok := terminals[tc.Name]
			//only create new terminal if it doesn't exist yet
			if !ok {
				newTerminal = &terminal{
					name:  tc.Name,
					index: tc.Index,
				}
				terminals[tc.Name] = newTerminal
			}

			newCombination.terminals[tc.Name] = newTerminal
			// we append the name since each combination is a concatenation of
			// all terminal names (e.g. OB114L20M3LEU)
			tNames = append(tNames, tc.Name)
		}

		// to maintain consistency we always sort the terminal names alphabetically
		s.combinations[sortStrings(tNames...)] = newCombination

	}

	// copy terminals from the de-duplicated map in our main data structure as a slice
	for _, t := range terminals {
		s.terminals = append(s.terminals, t)
	}

	// copy pins from the de-duplicated map in our main data structure as a slice
	for _, p := range pins {
		s.pins = append(s.pins, p)

		// deactivate all pins in startup
		if p.setState(false); err != nil {
			return err
		}
	}

	return nil
}

// sortStrings sorts the provided strings alphabetically and returns
// them as a single concatenated string
func sortStrings(strs ...string) string {
	sort.Slice(strs, func(i, j int) bool {
		return strs[i] < strs[j]
	})
	jointStrs := ""
	for _, str := range strs {
		jointStrs += str
	}
	return jointStrs
}

func (s *SmGPIO) Name() string {
	s.RLock()
	defer s.RUnlock()
	return s.name
}

func (s *SmGPIO) SetPort(req sw.Port) error {
	s.Lock()
	defer s.Unlock()

	// in tNames we store the current / to be modified states of our terminals
	tNames := make(map[string]bool, len(s.terminals))

	// get the current state of our terminals
	for _, t := range s.terminals {
		tNames[t.name] = t.state
	}

	// add the requested state of the terminals
	for _, t := range req.Terminals {

		// make sure the requested terminals exist
		if _, ok := tNames[t.Name]; !ok {
			return fmt.Errorf("unknown terminal %s", t.Name)
		}

		// set the requested state
		tNames[t.Name] = t.State
	}

	// identify the terminals which have to be set
	names := []string{}
	for tName := range tNames {
		if tNames[tName] {
			names = append(names, tName)
		}
	}

	// get the combination which corresponds to these terminals to be set
	c, ok := s.combinations[sortStrings(names...)]
	if !ok {
		return fmt.Errorf("unknown terminal combination")
	}

	// deactivate everything
	for _, r := range s.pins {
		r.setState(false)
	}
	for _, t := range s.terminals {
		t.state = false
	}

	// activate the relays of the new combination
	for _, r := range c.relays {
		r.setState(true)
	}

	// set the state of the terminals of the new combination
	for _, t := range c.terminals {
		t.state = true
	}

	// notify the listener that something has changed
	if s.eventHandler != nil {
		go s.eventHandler(s, s.serialize())
	}

	return nil
}

func (r *pin) setState(state bool) error {

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

func (s *SmGPIO) GetPort(portName string) (sw.Port, error) {
	s.RLock()
	defer s.RUnlock()

	p := sw.Port{
		Name:      s.portName,
		Index:     0,
		Terminals: s.serializeTerminals(),
	}

	return p, nil
}

func (s *SmGPIO) serializeTerminals() []sw.Terminal {

	terminals := []sw.Terminal{}

	for _, term := range s.terminals {
		t := sw.Terminal{
			Name:  term.name,
			Index: term.index,
			State: term.state,
		}
		terminals = append(terminals, t)
	}

	// sort the Terminals by index
	sort.Slice(terminals, func(i, j int) bool {
		return terminals[i].Index < terminals[j].Index
	})

	return terminals
}

func (s *SmGPIO) Serialize() sw.Device {
	s.RLock()
	defer s.RUnlock()

	return s.serialize()
}

func (s *SmGPIO) serialize() sw.Device {

	dev := sw.Device{
		Name:  s.name,
		Index: s.index,
		Ports: []sw.Port{
			sw.Port{
				Name:      s.portName,
				Index:     0, // fixed, since a stackmatch only has one port
				Terminals: s.serializeTerminals(),
			},
		},
	}
	return dev
}

func (s *SmGPIO) Close() {
	// nothing to do
}
