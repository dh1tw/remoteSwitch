//go:generate stringer -type=rbModel

package remotebox

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	sw "github.com/dh1tw/remoteSwitch/switch"
	"github.com/tarm/serial"
)

type rbModel int

// EA4TX added new models
const (
	rbUnknown rbModel = iota
	rb1x6
	rb2x6
	rb1x8
	rb2x8
	rb4sq
	rb2x12
	rb4sqplus
	rbrelay
	rb1x3
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
	prefix  string
	modelID int
	ports   int
	ants    []rbAnt
}

// EA4TX ordered - changed ModelID & added new models
var rbConfigs = map[rbModel]rbConfig{
	rb1x6: rbConfig{
		prefix:  "SW",
		modelID: 1,
		ports:   1,
		ants:    []rbAnt{rbAnt1, rbAnt2, rbAnt3, rbAnt4, rbAnt5, rbAnt6},
	},
	rb2x6: rbConfig{
		prefix:  "SW",
		modelID: 2,
		ports:   2,
		ants:    []rbAnt{rbAnt1, rbAnt2, rbAnt3, rbAnt4, rbAnt5, rbAnt6},
	},
	rb1x8: rbConfig{
		prefix:  "SW",
		modelID: 3,
		ports:   1,
		ants:    []rbAnt{rbAnt1, rbAnt2, rbAnt3, rbAnt4, rbAnt5, rbAnt6, rbAnt7, rbAnt8},
	},
	rb2x8: rbConfig{
		prefix:  "SW",
		modelID: 4,
		ports:   2,
		ants:    []rbAnt{rbAnt1, rbAnt2, rbAnt3, rbAnt4, rbAnt5, rbAnt6, rbAnt7, rbAnt8},
	},
	rb4sq: rbConfig{
		prefix:  "SQ",
		modelID: 5,
		ports:   1,
		ants:    []rbAnt{rbAnt1, rbAnt2, rbAnt3, rbAnt4},
	},
	rb2x12: rbConfig{
		prefix:  "SW",
		modelID: 6,
		ports:   2,
		ants:    []rbAnt{rbAnt1, rbAnt2, rbAnt3, rbAnt4, rbAnt5, rbAnt6, rbAnt7, rbAnt8, rbAnt9, rbAnt10, rbAnt11, rbAnt12},
	},
	rb4sqplus: rbConfig{
		prefix:  "SQ",
		modelID: 7,
		ports:   1,
		ants:    []rbAnt{rbAnt1, rbAnt2, rbAnt3, rbAnt4},
	},
	rbrelay: rbConfig{
		prefix:  "SW",
		modelID: 8,
		ports:   1,
		ants:    []rbAnt{rbAnt1, rbAnt2, rbAnt3, rbAnt4, rbAnt5, rbAnt6, rbAnt7, rbAnt8},
	},
	rb1x3: rbConfig{
		prefix:  "ST",
		modelID: 9,
		ports:   1,
		ants:    []rbAnt{rbAnt1, rbAnt2, rbAnt3},
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
	name            string
	portName        string
	index           int
	model           rbModel
	firmwareVersion string
	ports           map[string]*port
	portsList       []*port
	Ipaddress       string
	Ipport          int
	Connection      int
	sp              io.ReadWriteCloser
	//spipaddress         string
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

func New(opts ...func(*Remotebox)) *Remotebox {

	r := &Remotebox{
		name:              "EA4TX Remotebox",
		model:             rbUnknown,
		ports:             make(map[string]*port),
		portsList:         []*port{},
		Ipaddress:         "127.0.0.1",
		Ipport:            6000,
		Connection:        0, // 0: serial  1:tcp/ip
		spPollingInterval: time.Millisecond * 1000,
		spPortname:        "/dev/ttyACM0",
		spBaudrate:        9600, //doesn't really matter
		closeCh:           make(chan struct{}),
		errorCh:           make(chan struct{}),
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

func (r *Remotebox) Init() error {

	spConfig := &serial.Config{
		Name:        r.spPortname,
		Baud:        r.spBaudrate,
		ReadTimeout: time.Second,
		Parity:      serial.ParityNone,
		Size:        8,
		StopBits:    1,
	}

	if r.Connection == 0 { // serial
		sp, err := serial.OpenPort(spConfig)
		if err != nil {
			return err
		}
		r.sp = sp
	} else { // TCP
		sp, err := net.Dial("tcp", fmt.Sprintf("%s"+":"+"%d", r.Ipaddress, r.Ipport))
		if err != nil {
			return err
		}
		r.sp = sp
	}

	r.spReader = bufio.NewReader(r.sp)

	deviceInfo, err := r.getDeviceInfo()
	if err != nil {
		log.Panic(err)
	}

	model, fwVersion, err := parseDeviceInfo(deviceInfo)
	if err != nil {
		log.Panic(err)
	}

	r.model = model
	r.firmwareVersion = fwVersion

	config, err := r.getConfig()
	if err != nil {
		log.Panic(err)
	}

	eMap, err := parseConfig(config)
	if err != nil {
		log.Panic(err)
	}

	log.Printf("found ea4tx remotebox model: %s, firmware: %s", r.model, r.firmwareVersion)

	err = r.createPorts(model, fwVersion, eMap)
	if err != nil {
		log.Panic(err)
	}

	go r.start()

	return nil
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

// getDeviceInfo reads the device information (model and firmware)
// from the remotebox
func (r *Remotebox) getDeviceInfo() ([]string, error) {

	readError := fmt.Errorf("unable to read the device info from remotebox")

	_, err := r.write([]byte("O\n"))
	if err != nil {
		return nil, readError
	}

	line1, err := r.read()
	if err != nil {
		return nil, readError
	}

	line2, err := r.read()
	if err != nil {
		// buf in firmware versions < 1.3g. The do not send the
		// first line (EA4TX - model) but rather just an empty one
		// if err != io.EOF {
		// 	return nil, readError
		// }
	}

	return []string{line1, line2}, nil
}

func parseDeviceInfo(deviceInfo []string) (rbModel, string, error) {

	rbModel := rbUnknown
	parseError := fmt.Errorf("unable to parse remotebox device info")

	if deviceInfo == nil {
		return rbModel, "", parseError
	}

	// remotebox device info should contain two lines:
	// EA4TX AS2x12
	// Ver1.3g Firm:610

	// however in firmware < 1.3g this first line is empty

	if len(deviceInfo) != 2 {
		return rbModel, "", parseError
	}

	// version is an iterims slice of strings containing the remotebox model version
	// and the firmware
	version := []string{}

	// since remotebox firmware < 1.3g omits the first line
	// the version is now in the first slice while the second
	// one only contains one whitespace character
	if len(deviceInfo[1]) <= 1 {
		version = strings.Fields(deviceInfo[0])
		// firmware >= 1.3g sends two lines
	} else {
		// split the second line into two slices
		version = strings.Fields(deviceInfo[1])
	}

	// make sure we get two slices
	if len(version) != 2 {
		return rbModel, "", parseError
	}

	// determine the firmware version
	fwVersion := strings.ToLower(version[0][3:])

	// determine the model
	m, err := strconv.Atoi(version[1][5:6])
	if err != nil {
		return rbModel, "", parseError
	}

	for rbm, c := range rbConfigs {
		if c.modelID == m {
			rbModel = rbm
			break
		}
	}

	if rbModel == rbUnknown {
		return rbModel, fwVersion, fmt.Errorf("unsupported remotebox model")
	}

	switch fwVersion {
	case "1.3d", "1.3b", "1.2p", "1.2l":
		//pass
	// starting from 1.3g remotebox always identifies with "EA4TX" in the first line
	default:
		if !strings.Contains(deviceInfo[0], "EA4TX") {
			return rbModel, "", parseError
		}
	}

	return rbModel, fwVersion, nil
}

func (r *Remotebox) getConfig() (string, error) {

	readError := fmt.Errorf("unable to read config from remotebox")

	_, err := r.write([]byte("FI\n"))
	if err != nil {
		return "", readError
	}

	config := ""

	for i := 0; i <= 15; i++ {
		c, err := r.read()
		if err != nil {
			//		if err == io.EOF {
			//			continue
			//		}
			return "", err
		}
		config = config + c
	}

	return config, nil
}

// Start the main event loop for the serial port.
// It will query the remotebox for the current state of the port(s)
// with the pollingrate defined during initialization.
// If an error occures, the errorCh will be closed.
// Consequently the communication will be shut down and the object
// prepared for garbage collection.
func (r *Remotebox) start() {
	defer r.Close()

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

		// this is a blocking function which will run timeout
		// when no data is received (ReadTimeout)
		msg, err := r.read()
		if err != nil {
			if err == io.EOF {
				continue
			}
			fmt.Printf("serial port read error (%s on %s): %s\n",
				r.name, r.spPortname, err)
			close(r.errorCh)
			return // exit
		}
		r.resetWatchdog()
		if err := r.parseMsg(msg); err != nil {
			log.Println(err)
		}
	}
}

// poll the Remotebox for the current state
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
				fmt.Println("communication lost with remotebox")
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
	msg = strings.Replace(msg, "\n", "", -1)
	msg = strings.Replace(msg, "\r", "", -1)

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

	stateChanged := false

	switch msg[0:2] {
	case "SW", "ST":
		p, ok := r.ports[msg[0:3]]
		if !ok {
			return fmt.Errorf("unknown port")
		}
		if p == nil {
			return fmt.Errorf("port does not exist")
		}

		msg = msg[5 : len(msg)-1]
		state := strings.Split(msg, ",")

		if p.terminals == nil {
			return fmt.Errorf("terminals do not exist")
		}

		if len(state) != len(p.terminalsList) {
			return fmt.Errorf("message content does not match with remotebox model")
		}

		for i := 0; i < len(state); i++ {

			newstate := false
			switch state[i] {
			case "1":
				newstate = true
			case "0":
				newstate = false
			default:
				return fmt.Errorf("unknown state of terminal")
			}

			if p.terminalsList[i].state != newstate {
				p.terminalsList[i].state = newstate
				stateChanged = true
			}
		}

	// assuming the the special 4SQ version "ST" behaves the same
	// way as the standard 4SQ version.
	case "SQ":
		portMsg := msg[0:3]
		p, ok := r.ports[portMsg]

		if !ok {
			return fmt.Errorf("unknown port %s", portMsg)
		}
		if p == nil {
			return fmt.Errorf("port %s does not exist", portMsg)
		}

		state, err := strconv.Atoi(msg[5 : len(msg)-1])
		if err != nil {
			return fmt.Errorf("invalid state message: %v", msg)
		}

		if p.terminals == nil {
			return fmt.Errorf("terminals do not exist for port %s", portMsg)
		}

		if state > len(p.terminalsList) {
			return fmt.Errorf("message content does not match with remotebox model")
		}

		// check if something has changed
		if p.terminalsList[state-1].state {
			return nil
		}

		// set all terminals to false
		for i := 0; i < len(p.terminalsList); i++ {
			p.terminalsList[i].state = false
		}

		// set the newly selected terminal
		p.terminalsList[state-1].state = true

		stateChanged = true

	default:
		// ignore
	}

	if stateChanged && r.eventHandler != nil {
		go r.eventHandler(r, r.serialize())
	}

	return nil
}

// parseConfig parses the configuration read from the remotebox
// and returns a map[string]:byte containing for each address
// the corresponing byte.
func parseConfig(config string) (map[string]byte, error) {

	cMap := make(map[string]byte)
	parseError := fmt.Errorf("unable to parse config of remotebox")

	//remove trailing whitespace
	config = strings.TrimSuffix(config, " ")

	// split the string up into tuples Address:Content (e.g. 7F:0F)
	tuples := strings.Split(config, " ")

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
		cMap[tuple[0]] = byte(i)
	}

	return cMap, nil
}

// createPorts initializes the datastructure for the particular remotebox
// model.
func (r *Remotebox) createPorts(model rbModel, fw string, eMap map[string]byte) error {

	r.Lock()
	defer r.Unlock()

	if model == rbUnknown {
		return fmt.Errorf("unknown remotebox model %s", model)
	}

	// get the configuration for this particular remotebox model
	config, ok := rbConfigs[model]
	if !ok {
		return fmt.Errorf("no configuration found for the remotebox model %s", model)
	}

	// create the ports
	for i := 0; i < config.ports; i++ {

		p := &port{
			name:          fmt.Sprintf("%v%d", config.prefix, i+1),
			index:         i + 1,
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
				index: j + 1,
				name:  tName,
			}
			p.terminals[tName] = t
			p.terminalsList = append(p.terminalsList, t)
		}

		r.ports[p.name] = p
		r.portsList = append(r.portsList, p)
	}

	return nil
}

// getTerminalName is a helper function which returns the name of a terminal
// based on the values read from the remotebox's configuration. With this, the
// terminal names will match the ones displayed on the remotebox's LCD.
func getTerminalName(eMap map[string]byte, ant rbAnt) (string, error) {

	name := ""
	for _, addr := range rbAnts[ant] {
		c, ok := eMap[addr]
		if !ok {
			return "", fmt.Errorf("unknown remotebox config address: %s", addr)
		}
		name = name + string(c)
	}

	return name, nil
}

// Name returns the Name of this remotebox
func (r *Remotebox) Name() string {
	r.RLock()
	defer r.RUnlock()
	return r.name
}

// GetPort returns switch.Port struct containing the current state of
// the requested port.
func (r *Remotebox) GetPort(portname string) (sw.Port, error) {

	r.RLock()
	defer r.RUnlock()

	p, ok := r.ports[portname]
	if !ok {
		return sw.Port{}, fmt.Errorf("%s is an invalid port", portname)
	}

	return p.serialize(), nil
}

// SetPort sets the Terminals of a particular Port. The portRequest
// can contain n termials.
func (r *Remotebox) SetPort(req sw.Port) error {
	r.Lock()
	defer r.Unlock()

	// ensure that the requested port exists
	p, ok := r.ports[req.Name]
	if !ok {
		return fmt.Errorf("%s is an invalid port", req.Name)
	}

	// ensure that the requested terminal exists
	for n, t := range req.Terminals {
		rbTerminal, ok := p.terminals[t.Name]
		// copy the index of the terminal as it is not supplied with the
		// RPC request
		req.Terminals[n].Index = rbTerminal.index
		if !ok {
			return fmt.Errorf("%s is an invalid terminal", t.Name)
		}
	}

	for _, t := range req.Terminals {
		// Remotebox does not allow to unset a port. So we
		// ignore state==false
		if !t.State {
			continue
		}
		cmd := fmt.Sprintf("%dR", p.index)
		switch t.Index {
		case 10:
			cmd = fmt.Sprintf("%sA1\n", cmd)
		case 11:
			cmd = fmt.Sprintf("%sB1\n", cmd)
		case 12:
			cmd = fmt.Sprintf("%sC1\n", cmd)
		default:
			cmd = fmt.Sprintf("%s%d1\n", cmd, t.Index)
		}
		r.write([]byte(cmd))
	}

	return nil
}

// Serialize returns a switch.Device struct containing the current
// state and configuration of this Remotebox.
func (r *Remotebox) Serialize() sw.Device {
	r.RLock()
	defer r.RUnlock()

	return r.serialize()
}

// String returns the port with all of it's terminals and their
// corresponding state in a print friendly string.
func (p *port) String() string {
	s := fmt.Sprintf("%v: \n", p.name)
	for _, t := range p.terminalsList {
		s = s + fmt.Sprintf("  %v:%v\n", t.name, t.state)
	}

	return s
}

// serialize returns a switch.Device struct containing the current
// state and configuration of this remotebox. This method
// is not threadsafe.
func (r *Remotebox) serialize() sw.Device {

	dev := sw.Device{
		Name:  r.name,
		Index: r.index,
	}

	// serialize all ports
	for _, p := range r.portsList {
		swPort := p.serialize()
		dev.Ports = append(dev.Ports, swPort)
	}

	return dev
}

// serialize returns a switch.Port struct containing the current
// state and configuration of this port. This method is not threadsafe.
func (p *port) serialize() sw.Port {
	swPort := sw.Port{
		Name:      p.name,
		Index:     p.index,
		Terminals: []sw.Terminal{},
	}

	for _, r := range p.terminalsList {
		t := sw.Terminal{
			Name:  r.name,
			Index: r.index,
			State: r.state,
		}
		swPort.Terminals = append(swPort.Terminals, t)
	}

	return swPort
}
