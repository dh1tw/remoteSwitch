package hub

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	sw "github.com/dh1tw/remoteSwitch/switch"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

func (hub *Hub) wsHandler(w http.ResponseWriter, r *http.Request) {

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	c := &WsClient{
		Conn: conn,
	}

	hub.RLock()
	for _, r := range hub.switches {
		ev := Event{
			Name:       AddSwitch,
			DeviceName: r.Name(),
		}
		if err := c.write(ev); err != nil {
			fmt.Println(err)
		}
	}
	hub.RUnlock()

	hub.addWsClient(c)
}

func (hub *Hub) switchesHandler(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	s := hub.serializeSwitches()

	if err := json.NewEncoder(w).Encode(s); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("unable to encode switches msg"))
	}
}

func (hub *Hub) switchHandler(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	vars := mux.Vars(req)
	sName := vars["switch"]

	s, ok := hub.Switch(sName)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("unable to find switch"))
		return
	}

	if err := json.NewEncoder(w).Encode(s.Serialize()); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("unable to encode switch data to json"))
	}
}

func (hub *Hub) portHandler(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	vars := mux.Vars(req)
	sName := vars["switch"]
	sPort := vars["port"]

	s, ok := hub.Switch(sName)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("unable to find switch"))
		return
	}

	p, err := s.GetPort(sPort)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("unable to find port"))
		return
	}

	switch req.Method {
	case "GET":
		if err := json.NewEncoder(w).Encode(p); err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("unable to encode port to json"))
		}

	case "PUT":
		p := sw.Port{}
		dec := json.NewDecoder(req.Body)

		if err := dec.Decode(&p); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("invalid json"))
			return
		}

		if len(p.Name) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("invalid request"))
			return
		}

		if len(p.Terminals) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("invalid request"))
			return
		}

		err := s.SetPort(p)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("unable to set port %s: %s", p.Name, err)))
			return

		}

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}

}

func (hub *Hub) terminalHandler(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	vars := mux.Vars(req)
	sName := vars["switch"]
	sPort := vars["port"]
	sTerminal := vars["terminal"]

	s, ok := hub.Switch(sName)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("unable to find switch"))
		return
	}

	p, err := s.GetPort(sPort)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("unable to find port"))
		return
	}

	var t sw.Terminal

	for _, terminal := range p.Terminals {
		if terminal.Name == sTerminal {
			t = terminal
			break
		}
	}

	if t.Name == "" {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("unable to find terminal"))
		return
	}

	switch req.Method {
	case "GET":
		if err := json.NewEncoder(w).Encode(t); err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("unable to encode terminal to json"))
		}

	case "PUT":
		t := sw.Terminal{}
		dec := json.NewDecoder(req.Body)

		if err := dec.Decode(&t); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("invalid json"))
			return
		}

		if len(t.Name) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("invalid request"))
			return
		}

		portReq := p
		portReq.Terminals = []sw.Terminal{t}

		err := s.SetPort(portReq)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("unable to set terminal %s on port %s: %s", t.Name, p.Name, err)))
			return
		}

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}

}

func (hub *Hub) serializeSwitches() map[string]sw.Device {

	hub.RLock()
	defer hub.RUnlock()

	rs := make(map[string]sw.Device)

	for _, r := range hub.switches {
		sr := r.Serialize()
		rs[sr.Name] = sr
	}

	return rs
}
