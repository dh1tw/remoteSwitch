package ip9258

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"sync"
	"time"

	sw "github.com/dh1tw/remoteSwitch/switch"
)

type IP9258 struct {
	sync.RWMutex
	name                 string
	index                int
	portName             string
	pollingInterval      time.Duration
	pollingTicker        *time.Ticker
	terminals            map[int]*Terminal
	username             string
	password             string
	rawurl               string
	url                  string //includes username & password
	terminalStatePattern *regexp.Regexp
	eventHandler         func(sw.Switcher, sw.Device)
	closer               sync.Once
	stopPolling          chan struct{}
	errorCh              chan struct{}
}

// Terminal is the smallest unit and typically is something that is
// switched by one relay.
type Terminal struct {
	Name   string // A name associated to the terminal
	Outlet int    // Outlet is the physical outlet/plug/terminal
	Index  int    // Index sets the order in which it will be dispplayed on the GUI
	state  bool
}

func NewIP9258(options ...func(*IP9258)) *IP9258 {

	d := &IP9258{
		name:            "myIP9258",
		index:           0,
		portName:        "PS",
		username:        "admin",    //IP9258 default username
		password:        "12345678", //IP92584 default password
		rawurl:          "192.168.10.10",
		pollingInterval: time.Second * 3,
		terminals: map[int]*Terminal{
			1: &Terminal{Name: "AC Outlet 1", Outlet: 1, Index: 1, state: false},
			2: &Terminal{Name: "AC Outlet 2", Outlet: 2, Index: 2, state: false},
			3: &Terminal{Name: "AC Outlet 3", Outlet: 3, Index: 3, state: false},
			4: &Terminal{Name: "AC Outlet 4", Outlet: 4, Index: 4, state: false},
		},
	}

	for _, opt := range options {
		opt(d)
	}

	r, _ := regexp.Compile("p6[1-4]=[0-1]")

	d.terminalStatePattern = r

	d.url = createDeviceURL(d.username, d.password, d.rawurl)

	return d
}

func (d *IP9258) Init() error {
	d.Lock()
	defer d.Unlock()

	// make a first query to ensure that the device is actually
	// reachable and that the credentials are correct
	resp, err := d.queryTerminalStatus()
	if err != nil {
		return err
	}

	if err := d.updateTerminals(resp); err != nil {
		return err
	}

	d.pollingTicker = time.NewTicker(d.pollingInterval)
	d.stopPolling = make(chan struct{})

	go d.poll()

	return nil
}

// Close shuts down the switch.
func (d *IP9258) Close() {
	d.Lock()
	defer d.Unlock()

	if d.pollingTicker != nil {
		d.pollingTicker.Stop()
	}
	d.closer.Do(func() {
		d.stopPolling <- struct{}{}
	})
}

// poll the IP9258 webswitch for the current status of the terminals. In case the
// the webswitch has been modified either manually or through the web interface, we
// have to bring remoteRotator again back in sync.
// This function is blocking and executes and infinite loop. It should be executed
// in it's own go routine.
func (d *IP9258) poll() {
	for {
		select {
		case <-d.pollingTicker.C:
			d.Lock()
			res, err := d.queryTerminalStatus()
			if err != nil {
				log.Println(err)
			}
			if err := d.updateTerminals(res); err != nil {
				log.Println(err)
			}
			d.Unlock()
		case <-d.stopPolling:
			return
		}
	}
}

// Name returns the Name of this Dummy Switch
func (d *IP9258) Name() string {
	d.RLock()
	defer d.RUnlock()
	return d.name
}

// SetPort sets the Terminals of a particular Port. The portRequest
// can contain n termials.
func (d *IP9258) SetPort(portRequest sw.Port) error {
	d.Lock()
	defer d.Unlock()

	for _, treq := range portRequest.Terminals {
		t, err := d.getTerminal(treq.Name)
		if err != nil {
			continue
		}
		if err := d.setTerminal(t.Outlet, treq.State); err != nil {
			return err
		}
	}

	return nil
}

// GetPort returns switch.Port struct containing the current state of
// the port. Portname is ignored since this device will only ever
// contain one port.
func (d *IP9258) GetPort(portName string) (sw.Port, error) {
	d.RLock()
	defer d.RUnlock()

	return d.getPort(), nil
}

func (d *IP9258) getPort() sw.Port {

	p := sw.Port{
		Name:      d.portName,
		Index:     0, //this type of Switch only has one Port ever
		Terminals: []sw.Terminal{},
	}

	for _, t := range d.terminals {
		swt := sw.Terminal{
			Name:  t.Name,
			Index: t.Index,
			State: t.state,
		}
		p.Terminals = append(p.Terminals, swt)
	}

	// Sort the slice of Terminals by index
	sort.Slice(p.Terminals, func(i, j int) bool {
		return p.Terminals[i].Index < p.Terminals[j].Index
	})

	return p
}

// Serialize returns a switch.Device struct containing the current
// state and configuration of this IP9258 power switch.
func (d *IP9258) Serialize() sw.Device {
	d.RLock()
	defer d.RUnlock()

	return d.serialize()
}

// serialize returns a switch.Device struct containing the current
// state and configuration of this Dummy switch. This method
// is not threadsafe.
func (d *IP9258) serialize() sw.Device {

	device := sw.Device{
		Name:  d.name,
		Index: d.index,
		Ports: []sw.Port{
			d.getPort(),
		},
	}

	return device
}

// getTerminal returns the pointer to a Terminal requested by it's name.
// If no Terminal is found under the specified name, nil and an error
// will be returned.
func (d *IP9258) getTerminal(name string) (*Terminal, error) {

	for _, t := range d.terminals {
		if t.Name == name {
			return t, nil
		}
	}

	return nil, fmt.Errorf("terminal %v does not exist", name)
}

// queryTerminalStatus makes an HTTP call to the IP9258 and requests the
// status of the terminals. The method returns the raw string from the
// IP9258's response body.
func (s *IP9258) queryTerminalStatus() (string, error) {

	client := http.Client{
		Timeout: 3 * time.Second,
	}

	resp, err := client.Get(fmt.Sprintf("%v/set.cmd?cmd=getpower", s.url))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%v: unable to query ip9258", resp.StatusCode)
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(respBody), nil
}

// setTerminal wraps an HTTP call to activate or deactivate a particular terminal
// of the IP9258.
func (s *IP9258) setTerminal(terminal int, newstate bool) error {

	newState := "0"
	if newstate {
		newState = "1"
	}

	url := fmt.Sprintf("%v/set.cmd?cmd=setpower+p6%d=%v", s.url, terminal, newState)

	client := http.Client{
		Timeout: 3 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%v: unable to set terminal state of ip9258", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return s.updateTerminals(string(body))
}

// updateTerminals parses the string returned by the IP9258 containing the status
// of one or more terminals. This method will update the internal state and fire
// the eventhandler in case the state of at least one terminal has changed.
func (s *IP9258) updateTerminals(respBody string) error {

	res := s.terminalStatePattern.FindAllString(respBody, -1)

	if len(res) == 0 {
		return fmt.Errorf("provided input string '%s' does not contain any valid terminals", respBody)
	}

	updated := false

	for _, p := range res {
		terminalNo, state, err := parseTerminalState(p)
		if err != nil {
			return err
		}
		updated, err = s.updateTerminal(terminalNo, state)
		if err != nil {
			return err
		}
	}

	if updated && s.eventHandler != nil {
		device := s.serialize()
		go s.eventHandler(s, device)
	}

	return nil
}

// updateTerminal checks the if the state of a terminal has changed and
// updated its if necessary. If changed, the function returns true
func (s *IP9258) updateTerminal(terminal int, newState bool) (bool, error) {
	t, ok := s.terminals[terminal]
	if !ok {
		return false, fmt.Errorf("unknown terminal %v", terminal)
	}
	if t.state == newState { // no need to update
		return false, nil
	}
	t.state = newState
	return true, nil
}

// parseTerminalState parses a string containing the state of one of
// the IP9258's terminals. The expected format is "p6x=y" where x is
// the terminal enumeration (1...4) and y it's state (0 or 1).
// This function returns the terminal enumeration (int), it's state (bool)
// or an error if the string couldn't be parsed successfully.
func parseTerminalState(input string) (int, bool, error) {

	t, err := strconv.Atoi(string(input[2]))
	if err != nil {
		return 0, false, err
	}
	var state bool
	switch input[4] {
	case '0':
		state = false
	case '1':
		state = true
	default:
		return t, false, fmt.Errorf("unknown state")
	}

	return t, state, nil
}

// createDeviceURL assembles the IP9258 connection string which
// embeds the username and password into the URL.
func createDeviceURL(username, password, url string) string {

	return fmt.Sprintf("http://%s:%s@%s",
		username,
		password,
		url)
}
