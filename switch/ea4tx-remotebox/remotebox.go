//go:generate stringer -type=rbModel

package remotebox

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
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
)

type Remotebox struct {
	sync.RWMutex
	name              string
	portName          string
	index             int
	model             rbModel
	firmwareVersion   string
	sp                io.ReadWriteCloser
	spPortname        string
	spBaudrate        int
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
		spPollingInterval: time.Second * 1,
		spPortname:        "/dev/ttyACM0",
		spBaudrate:        9600,
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

	// create the necessary structures

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

	spReader := bufio.NewReader(r.sp)

read:
	msg, err := spReader.ReadString('\n')
	if err != nil {
		return string(msg), err
	}
	msg = strings.ReplaceAll(msg, "\n", "")
	msg = strings.ReplaceAll(msg, "\r", "")

	if len(msg) == 0 {
		// this is a hack, since the Remotebox sends '\n\r' at the beginning
		// of the string containing the configuration ("S"). This lead to
		// ocassional loss of the first characters ("SW1:..."). Apparently
		// it takes too long to re-initalize a new Reader in the next
		// loop cycle. Therefore empty messages (having \n\r stripped off)
		// will remain in this routing and jump back to the 'read' label.
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
func (r *Remotebox) parseMsg(msg string) {

	if len(msg) == 0 {
		return
	}

	fmt.Println("msg:", msg)
	for _, char := range msg {
		fmt.Printf("0x%x ", char)
	}
	fmt.Printf("\n")
	fmt.Println("")
}

func (r *Remotebox) parseFirmware(s string) {

}

func (r *Remotebox) parseSwitch(s string) {

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

func (r *Remotebox) Serialize() sw.Device {

	s := sw.Device{}
	return s
}
