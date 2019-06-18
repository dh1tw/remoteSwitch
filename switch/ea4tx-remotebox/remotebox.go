//go:generate stringer -type=rbModel

package remotebox

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	sw "github.com/dh1tw/remoteSwitch/switch"
	"github.com/tarm/serial"
)

type rbModel int

const (
	rbUnknown rbModel = iota
	rb1x6
	rb2x6
	rb1x8
	rb2x8
	rb2x12
	rb4sq
	rb4sqplus
)

type rbAnt int

const (
	rbAnt1 rbAnt = iota
	rbAnt2
	rbAnt3
	rbAnt4
	rbAnt5
	rbAnt6
	rbAnt7
	rbAnt8
	rbAnt9
	rbAnt10
	rbAnt11
	rbAnt12
)

var rbAnts = map[rbAnt][]string{
	rbAnt1:  []string{"10", "11", "12", "13"},
	rbAnt2:  []string{"14", "15", "16", "17"},
	rbAnt3:  []string{"18", "19", "1A", "1B"},
	rbAnt4:  []string{"1C", "1D", "1E", "1F"},
	rbAnt5:  []string{"20", "21", "22", "23"},
	rbAnt6:  []string{"24", "25", "26", "27"},
	rbAnt7:  []string{"28", "29", "2A", "2B"},
	rbAnt8:  []string{"2C", "2D", "2E", "2F"},
	rbAnt9:  []string{"46", "47", "48", "49"},
	rbAnt10: []string{"4A", "4B", "4C", "4D"},
	rbAnt11: []string{"4E", "4F", "50", "51"},
	rbAnt12: []string{"52", "53", "54", "55"},
}

type rbConfig struct {
	name  string
	ports int
	ants  []rbAnt
}

var rbConfigs = map[rbModel]rbConfig{
	rb1x6: rbConfig{
		name:  "SW",
		ports: 1,
		ants:  []rbAnt{rbAnt1, rbAnt2, rbAnt3, rbAnt4, rbAnt5, rbAnt6},
	},
	rb2x6: rbConfig{
		name:  "SW",
		ports: 2,
		ants:  []rbAnt{rbAnt1, rbAnt2, rbAnt3, rbAnt4, rbAnt5, rbAnt6},
	},
	rb1x8: rbConfig{
		name:  "SW",
		ports: 1,
		ants:  []rbAnt{rbAnt1, rbAnt2, rbAnt3, rbAnt4, rbAnt5, rbAnt6, rbAnt7, rbAnt8},
	},
	rb2x8: rbConfig{
		name:  "SW",
		ports: 2,
		ants:  []rbAnt{rbAnt1, rbAnt2, rbAnt3, rbAnt4, rbAnt5, rbAnt6, rbAnt7, rbAnt8},
	},
	rb2x12: rbConfig{
		name:  "SW",
		ports: 2,
		ants:  []rbAnt{rbAnt1, rbAnt2, rbAnt3, rbAnt4, rbAnt5, rbAnt6, rbAnt7, rbAnt8, rbAnt9, rbAnt10, rbAnt11, rbAnt12},
	},
	rb4sq: rbConfig{
		name:  "SQ",
		ports: 1,
		ants:  []rbAnt{rbAnt1, rbAnt2, rbAnt3, rbAnt4},
	},
	rb4sqplus: rbConfig{
		name:  "ST",
		ports: 1,
		ants:  []rbAnt{rbAnt1, rbAnt2, rbAnt3, rbAnt4, rbAnt5, rbAnt6},
	},
}

type port struct {
	name          string
	index         int
	terminals     map[string]*terminal
	terminalsList []*terminal
}

type terminal struct {
	name  string
	index int
	state bool
}

type Remotebox struct {
	sync.RWMutex
	name              string
	portName          string
	index             int
	model             rbModel
	firmwareVersion   string
	ports             map[string]*port
	sp                io.ReadWriteCloser
	spPortname        string
	spBaudrate        int
	spReader          *bufio.Reader
	spRead            sync.Mutex
	spWrite           sync.Mutex
	spPollingInterval time.Duration
	spPollingTicker   *time.Ticker
	spWatchdogTs      time.Time
	eventHandler      func(sw.Switcher, sw.Device)
	closeCh           chan struct{}
	errorCh           chan struct{}
	starter           sync.Once
	closer            sync.Once
}

func New(opts ...func(*Remotebox)) (*Remotebox, error) {

	r := &Remotebox{
		name:              "EA4TX Remotebox",
		model:             rbUnknown,
		ports:             make(map[string]*port),
		spPollingInterval: time.Millisecond * 100,
		spPortname:        "/dev/ttyACM0",
		spBaudrate:        9600, //doesn't really matter
		closeCh:           make(chan struct{}),
	}

	for _, opt := range opts {
		opt(r)
	}

	spConfig := &serial.Config{
		Name:        r.spPortname,
		Baud:        r.spBaudrate,
		ReadTimeout: time.Second,
		Parity:      serial.ParityNone,
		Size:        8,
		StopBits:    1,
	}

	sp, err := serial.OpenPort(spConfig)
	if err != nil {
		return nil, err
	}

	r.sp = sp
	r.spReader = bufio.NewReader(r.sp)

	go r.start()

	return r, nil
}

func (r *Remotebox) Close() {
	r.Lock()
	r.spRead.Lock()
	r.spWrite.Lock()
	defer r.Unlock()
	defer r.spWrite.Unlock()
	defer r.spRead.Unlock()

	if r.spPollingTicker != nil {
		r.spPollingTicker.Stop()
	}
	// makes sure that the serial port and the event loop just gets closed once
	r.closer.Do(func() {
		close(r.closeCh)
		r.sp.Close()
	})
}

// resetWatchdog resets the watchdog. This means that a packet has been
// received from the Remotebox
func (r *Remotebox) resetWatchdog() {
	r.Lock()
	defer r.Unlock()
	r.spWatchdogTs = time.Now()
}

// checkWatchdog compares the watchdog timestamp with the current time
// and returns true if this value is greater than 5x updateInterval.
func (r *Remotebox) checkWatchdog() bool {
	r.Lock()
	defer r.Unlock()
	if time.Since(r.spWatchdogTs) > 5*r.spPollingInterval {
		return true
	}
	return false
}

func (r *Remotebox) getConfig() ([]string, error) {

	configError := errors.New("unable to read configuration from remotebox")

	_, err := r.write([]byte("O\n"))
	if err != nil {
		return nil, configError
	}

	line1, err := r.read()
	if err != nil {
		return nil, configError
	}

	line2, err := r.read()
	if err != nil {
		return nil, configError
	}

	return []string{line1, line2}, nil
}

func parseConfig(config []string) (rbModel, string, error) {

	rbModel := rbUnknown
	parseError := errors.New("unable to parse remotebox configuration")

	// remotebox config should contain two lines:
	// EA4TX AS2x12
	// Ver1.3g Firm:610

	if config == nil {
		return rbModel, "", parseError
	}

	if len(config) != 2 {
		return rbModel, "", parseError
	}

	// Remotebox always identifies with "EA4TX" in the first line
	if !strings.Contains(config[0], "EA4TX") {
		return rbModel, "", parseError
	}

	// split the second line into two slices
	version := strings.Fields(config[1])
	if len(version) != 2 {
		return rbModel, "", parseError
	}

	// determine the firmware version
	fwVersion := version[0][3:]

	// determine the model
	switch version[1][5:6] {
	case "1":
		rbModel = rb1x6
	case "2":
		rbModel = rb2x6
	case "3":
		rbModel = rb1x8
	case "4":
		rbModel = rb2x8
	case "5":
		rbModel = rb4sq
	case "6":
		rbModel = rb2x12
	default:
		return rbModel, fwVersion, errors.New("unsupported remotebox model")
	}

	return rbModel, fwVersion, nil
}

func (r *Remotebox) getEEPROM() (string, error) {

	eepromError := errors.New("unable to read EEPROM of remotebox")

	_, err := r.write([]byte("FI\n"))
	if err != nil {
		return "", eepromError
	}

	eeprom := ""

	for i := 0; i <= 15; i++ {
		c, err := r.read()
		if err != nil {
			if err == io.EOF {
				continue
			}
			return "", err
		}
		eeprom = eeprom + c
	}

	return eeprom, nil
}

// Start the main event loop for the serial port.
// It will query the Remotebox for the current status of the port(s)
// with the pollingrate defined during initialization.
// If an error occures, the errorCh will be closed.
// Consequently the communication will be shut down and the object
// prepared for garbage collection.
func (r *Remotebox) start() {
	defer r.Close()

	configRaw, err := r.getConfig()
	if err != nil {
		log.Panic(err)
	}

	model, fwVersion, err := parseConfig(configRaw)
	if err != nil {
		log.Panic(err)
	}

	r.model = model
	r.firmwareVersion = fwVersion

	eeprom, err := r.getEEPROM()
	if err != nil {
		log.Panic(err)
	}

	eMap, err := parseEEPROM(eeprom)
	if err != nil {
		log.Panic(err)
	}

	err = r.createPorts(model, fwVersion, eMap)
	if err != nil {
		log.Panic(err)
	}

	r.Lock()
	r.spPollingTicker = time.NewTicker(r.spPollingInterval)
	r.spWatchdogTs = time.Now()
	r.Unlock()

	// start async polling
	go r.poll()

	for {
		select {
		// when closing has been signaled, stop reading
		// from the serial port by exiting this function
		case <-r.closeCh:
			return
		default:
		}

		// this is a blocking function which will run eventually
		// into a timeout if no data is received
		msg, err := r.read()
		if err != nil {
			// serialport read is expected to timeout after 100ms
			// to unblock this routine
			if err == io.EOF {
				continue
			}
			fmt.Printf("serial port read error (%s on %s): %s\n",
				r.name, r.spPortname, err)
			close(r.errorCh)
			return // exit
		}
		r.resetWatchdog()
		r.parseMsg(msg)
	}
}

// poll the Remotebox rotator for the current heading (azimuth + elevation)
func (r *Remotebox) poll() {
	defer r.Close()

	for {
		select {
		case <-r.spPollingTicker.C:
			if err := r.query(); err != nil {
				fmt.Println("serial port write error:", err)
				close(r.errorCh)
				return
			}
			if r.checkWatchdog() {
				fmt.Println("communication lost with Remotebox")
				close(r.errorCh)
				return
			}
		// when closing has been signaled, stop polling and return
		case <-r.closeCh:
			return
		}
	}
}

// read from the Remotebox through this wrapper function
func (r *Remotebox) read() (string, error) {
	r.spRead.Lock()
	defer r.spRead.Unlock()

read:
	msg, err := r.spReader.ReadString('\n')
	if err != nil {
		return msg, err
	}

	// sanitize string by removing line breaking characters
	msg = strings.ReplaceAll(msg, "\n", "")
	msg = strings.ReplaceAll(msg, "\r", "")

	// discard empty lines
	if len(msg) == 0 {
		goto read
	}

	return msg, nil
}

// request the port status of the Remotebox
func (r *Remotebox) query() error {
	_, err := r.write([]byte("S\n"))
	return err
}

// all functions write to the Remotebox / serial port through this wrapper function
func (r *Remotebox) write(data []byte) (int, error) {
	r.spWrite.Lock()
	defer r.spWrite.Unlock()
	return r.sp.Write(data)
}

// parseMsg checks the content of the received message from the RemoteBox,
// updates the internal state and executes the event callback
func (r *Remotebox) parseMsg(msg string) error {
	r.Lock()
	defer r.Unlock()

	if len(msg) == 0 {
		return nil
	}

	// fmt.Println("msg:", msg)
	stateChanged := false

	switch msg[0:2] {
	case "SW":
		p, ok := r.ports[msg[0:3]]
		if !ok {
			return errors.New("unknown port")
		}
		if p == nil {
			return errors.New("port does not exist")
		}

		msg = msg[5 : len(msg)-1]
		state := strings.Split(msg, ",")

		if p.terminals == nil {
			return errors.New("terminals do not exist")
		}

		if len(state) != len(p.terminalsList) {
			return errors.New("message content does not match with remotebox model")
		}

		for i := 0; i < len(state); i++ {

			newstate := false
			switch state[i] {
			case "1":
				newstate = true
			case "0":
				newstate = false
			default:
				return errors.New("unknown state of terminal")
			}

			if p.terminalsList[i].state != newstate {
				p.terminalsList[i].state = newstate
				stateChanged = true
			}
		}

	case "SQ":

	case "ST":

	default:
		// do nothing
	}

	// if stateChanged && r.eventHandler != nil {
	if stateChanged {
		for _, p := range r.ports {
			fmt.Println(p)
		}
	}

	return nil
}

// parseEEPROM parses the configuration read from the remotebox
// eeprom and returns a map[string]:byte containing for each address
// the corresponing byte.
func parseEEPROM(eeprom string) (map[string]byte, error) {

	eMap := make(map[string]byte)
	parseError := errors.New("unable to parse EEPROM of remotebox")

	//remove trailing whitespace
	eeprom = strings.TrimSuffix(eeprom, " ")

	// split the string up into tuples Address:Content (e.g. 7F:0F)
	tuples := strings.Split(eeprom, " ")

	// write the tuples into a map
	for _, t := range tuples {
		tuple := strings.Split(t, ":")
		if len(tuple) != 2 {
			return nil, parseError
		}
		// convert the ASCII values (base 16 - hex) to integer
		i, err := strconv.ParseInt(tuple[1], 16, 0)
		if err != nil {
			return nil, parseError
		}
		eMap[tuple[0]] = byte(i)
	}

	return eMap, nil
}

// createPorts initializes the datastructure for the particular remotebox
// model.
func (r *Remotebox) createPorts(model rbModel, fw string, eMap map[string]byte) error {

	r.Lock()
	defer r.Unlock()

	if model == rbUnknown {
		return errors.New("unknown remotebox model")
	}

	// get the configuration for this particular remotebox model
	config, ok := rbConfigs[model]
	if !ok {
		return errors.New("no configuration found for this remotebox model")
	}

	// create the ports
	for i := 0; i < config.ports; i++ {

		p := &port{
			name:          fmt.Sprintf("%v%d", config.name, i+1),
			terminals:     make(map[string]*terminal),
			terminalsList: []*terminal{},
		}

		// create the terminals for the port
		for j := 0; j < len(config.ants); j++ {
			tName, err := getTerminalName(eMap, config.ants[j])
			if err != nil {
				return err
			}
			t := &terminal{
				index: j,
				name:  tName,
			}
			p.terminals[tName] = t
			p.terminalsList = append(p.terminalsList, t)
		}

		r.ports[p.name] = p
	}

	return nil
}

// getTerminalName is a helper function which returns the name of a terminal
// based on the values from the eeprom of the remotebox in order to match
// the names with the ones displayed on the LCD.
func getTerminalName(eMap map[string]byte, ant rbAnt) (string, error) {

	name := ""
	for _, addr := range rbAnts[ant] {
		c, ok := eMap[addr]
		if !ok {
			return "", errors.New("unknown remotebox eeprom address")
		}
		name = name + string(c)
	}

	return name, nil
}

func (r *Remotebox) Name() string {
	r.RLock()
	defer r.RUnlock()
	return r.name
}

func (r *Remotebox) GetPort(portname string) (sw.Port, error) {

	p := sw.Port{}
	return p, nil
}

func (r *Remotebox) SetPort(p sw.Port) error {
	return nil
}

func (r *Remotebox) Serialize() sw.Device {
	r.RLock()
	defer r.RUnlock()

	s := sw.Device{
		Name:  r.name,
		Index: r.index,
	}

	return s
}

func (p *port) String() string {
	s := fmt.Sprintf("%v: \n", p.name)
	for _, t := range p.terminalsList {
		s = s + fmt.Sprintf("  %v:%v\n", t.name, t.state)
	}

	return s
}
