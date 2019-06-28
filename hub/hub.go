package hub

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"sync"

	rice "github.com/GeertJohan/go.rice"
	sw "github.com/dh1tw/remoteSwitch/switch"
	"github.com/gorilla/mux"
)

// Hub is a struct which makes a switch available through http.
type Hub struct {
	sync.RWMutex
	wsClients     map[*WsClient]bool
	closeWsClient chan *WsClient
	router        *mux.Router
	fileServer    http.Handler
	switches      map[string]sw.Switcher
	apiVersion    string
	apiMatch      *regexp.Regexp
}

// NewHub returns the pointer to an initialized Hub object.
func NewHub(switches ...sw.Switcher) (*Hub, error) {
	hub := &Hub{
		wsClients:     make(map[*WsClient]bool),
		closeWsClient: make(chan *WsClient),
		switches:      make(map[string]sw.Switcher),
		apiVersion:    "1.0",
		apiMatch:      regexp.MustCompile(`api\/v\d\.\d\/`),
	}

	for _, r := range switches {
		if err := hub.AddSwitch(r); err != nil {
			return nil, err
		}
	}

	go hub.handleClose()

	return hub, nil
}

func (hub *Hub) handleClose() {
	for {
		select {
		case c := <-hub.closeWsClient:
			hub.removeWsClient(c)
		}
	}
}

// AddSwitch adds / registers a Switch. The switch's name must be unique.
func (hub *Hub) AddSwitch(r sw.Switcher) error {
	hub.Lock()
	defer hub.Unlock()

	return hub.addSwitch(r)
}

func (hub *Hub) addSwitch(r sw.Switcher) error {
	_, ok := hub.switches[r.Name()]
	if ok {
		return fmt.Errorf("the switch's names must be unique; %s provided twice", r.Name())
	}
	hub.switches[r.Name()] = r
	ev := Event{
		Name:       AddSwitch,
		DeviceName: r.Name(),
	}
	if err := hub.broadcastToWsClients(ev); err != nil {
		fmt.Println(err)
	}
	log.Printf("added switch (%s)\n", r.Name())

	return nil
}

// RemoveSwitch deletes / de-registers a switch.
func (hub *Hub) RemoveSwitch(r sw.Switcher) {
	hub.Lock()
	defer hub.Unlock()

	ev := Event{
		Name:       RemoveSwitch,
		DeviceName: r.Name(),
	}

	if err := hub.broadcastToWsClients(ev); err != nil {
		fmt.Println(err)
	}

	r.Close()
	delete(hub.switches, r.Name())
	log.Printf("removed switch (%s)\n", r.Name())
}

// Switch returns a particular Switch stored from the hub. If no
// Switch exists with that name, (nil, false) will be returned.
func (hub *Hub) Switch(name string) (sw.Switcher, bool) {
	hub.RLock()
	defer hub.RUnlock()

	s, ok := hub.switches[name]
	return s, ok
}

// Switches returns a slice of all registered Switches.
func (hub *Hub) Switches() []sw.Switcher {
	hub.RLock()
	defer hub.RUnlock()

	devices := make([]sw.Switcher, 0, len(hub.switches))
	for _, r := range hub.switches {
		devices = append(devices, r)
	}

	return devices
}

// AddWsClient registers a new websocket client
func (hub *Hub) addWsClient(client *WsClient) {
	hub.Lock()
	defer hub.Unlock()

	if _, alreadyInMap := hub.wsClients[client]; alreadyInMap {
		delete(hub.wsClients, client)
	}
	hub.wsClients[client] = true

	// we need to listen on the websocket so that the incoming ping
	// messages can be (automatically) answered (with a pong message)
	go client.listen(hub.closeWsClient)

	log.Printf("websocket client connected (%v)\n", client.RemoteAddr())
}

// removeWsClient removes a websocket client
func (hub *Hub) removeWsClient(c *WsClient) {
	hub.Lock()
	defer hub.Unlock()

	if _, ok := hub.wsClients[c]; ok {
		delete(hub.wsClients, c)
	}

	c.Close()
	log.Printf("websocket client disconnected (%v)\n", c.RemoteAddr())
}

// ListenHTTP starts a HTTP Server on a given network adapter / port and
// sets a HTTP and Websocket handler.
// Since this function contains an endless loop, it should be executed
// in a go routine. If the listener can not be initialized, it will
// close the errorCh channel.
func (hub *Hub) ListenHTTP(host string, port int, errorCh chan<- struct{}) {

	defer close(errorCh)

	box := rice.MustFindBox("../html")
	hub.fileServer = http.FileServer(box.HTTPBox())
	hub.router = mux.NewRouter().StrictSlash(true)

	// load the HTTP routes with their respective endpoints
	hub.routes()

	// Listen for incoming connections.
	log.Printf("listening on %s:%d for HTTP connections\n", host, port)

	err := http.ListenAndServe(fmt.Sprintf("%s:%d", host, port), hub.apiRedirectRouter(hub.router))
	if err != nil {
		log.Println(err)
		return
	}
}

type Event struct {
	Name       SwitchEvent `json:"name,omitempty"`
	DeviceName string      `json:"device_name,omitempty"`
	Device     sw.Device   `json:"device,omitempty"` //only used for updates
}

type SwitchEvent string

const (
	AddSwitch    SwitchEvent = "add"
	RemoveSwitch SwitchEvent = "remove"
	UpdateSwitch SwitchEvent = "update"
)

// Broadcast sends a rotator Status struct to all connected clients
func (hub *Hub) Broadcast(dev sw.Device) {

	ev := Event{
		Name:       UpdateSwitch,
		DeviceName: dev.Name,
		Device:     dev,
	}
	if err := hub.BroadcastToWsClients(ev); err != nil {
		log.Println(err)
	}
}

// BroadcastToWsClients will send a rotator.Status struct to all clients
// connected through a Websocket
func (hub *Hub) BroadcastToWsClients(event Event) error {
	hub.Lock()
	defer hub.Unlock()

	return hub.broadcastToWsClients(event)
}

func (hub *Hub) broadcastToWsClients(event Event) error {

	for c := range hub.wsClients {
		if err := c.write(event); err != nil {
			log.Printf("error writing to client %v: %v\n", c.RemoteAddr(), err)
			log.Printf("disconnecting client %v\n", c.RemoteAddr())
			c.Close()
			delete(hub.wsClients, c)
		}
	}

	return nil
}
