package hub

func (hub *Hub) routes() {
	hub.router.HandleFunc("/api/switches", hub.switchesHandler).Methods("GET")
	hub.router.HandleFunc("/api/switch/{switch}", hub.switchHandler).Methods("GET")
	hub.router.HandleFunc("/api/switch/{switch}/port/{port}", hub.portHandler)
	hub.router.HandleFunc("/api/switch/{switch}/port/{port}/terminal/{terminal}", hub.terminalHandler)
	hub.router.HandleFunc("/ws", hub.wsHandler)
	hub.router.PathPrefix("/").Handler(hub.fileServer)
}
