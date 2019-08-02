package hub

func (hub *Hub) routes() {
	hub.router.HandleFunc("/api/switches", hub.switchesHandler).Methods("GET")
	hub.router.HandleFunc("/api/switch/{switch}", hub.switchHandler).Methods("GET")
	hub.router.HandleFunc("/api/switch/{switch}/port", hub.switchPortHandler)
	hub.router.HandleFunc("/ws", hub.wsHandler)
	hub.router.PathPrefix("/").Handler(hub.fileServer)
}
