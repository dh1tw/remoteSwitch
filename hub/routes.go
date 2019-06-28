package hub

func (hub *Hub) routes() {
	// API v1.0
	hub.router.HandleFunc("/api/v1.0/switches", hub.switchesHandler).Methods("GET")
	hub.router.HandleFunc("/api/v1.0/switch/{switch}", hub.switchHandler).Methods("GET")
	hub.router.HandleFunc("/api/v1.0/switch/{switch}/port/{port}", hub.portHandler)
	hub.router.HandleFunc("/api/v1.0/switch/{switch}/port/{port}/terminal/{terminal}", hub.terminalHandler)

	hub.router.HandleFunc("/ws", hub.wsHandler)
	hub.router.PathPrefix("/").Handler(hub.fileServer)
}
