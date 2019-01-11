var vm = new Vue({
    el: '#app',

    data: {
        ws: null, // websocket
        Switches: {},
        hideConnectionMsg: false,
        resizeTimeout: null,
        connected: false,
    },
    components: {
        'sb-switch': Switch,

    },
    created: function () {
        window.addEventListener('resize', this.getWindowSize);
    },
    mounted: function () {
        this.openWebsocket();
    },
    beforeDestroy: function () {
        window.removeEventListener('resize', this.getWindowWidth);
    },
    methods: {

        // get the serialized switch object from the server
        getSwitchObj: function (switchName) {

            if (switchName in this.Switches) {
                return
            }

            this.$http.get("/api/switch/" + switchName).then(response => {
                this.addSwitch(response.body);
            })
        },

        // add a switch
        addSwitch: function (Switch) {

            if (!(Switch.name in this.Switches)) {
                this.$set(this.Switches, Switch.name, Switch);
            }
        },

        // remove a switch
        removeSwitch: function (switchName) {

            if (switchName in this.Switches) {

                this.$delete(this.Switches, switchName);
            }
        },

        // open the websocket and set an eventlister to receive updates
        // for switches
        openWebsocket: function () {
            var protocol = "ws://";
            if (window.location.protocol.indexOf("https") !== -1) {
                protocol = "wss://";
            }
            this.ws = new ReconnectingWebSocket(protocol + window.location.host + '/ws');
            this.ws.addEventListener('message', function (e) {
                var eventMsg = JSON.parse(e.data);
                console.log(eventMsg);

                // add switch
                if (eventMsg['name'] == 'add') {
                    this.getSwitchObj(eventMsg['device_name']);

                    // remove switch
                } else if (eventMsg['name'] == 'remove') {
                    this.removeSwitch(eventMsg['device_name']);

                    // update
                } else if (eventMsg['name'] == 'update') {
                    updatedDevice = eventMsg['device']
                    switchName = eventMsg['device_name']
                    if (switchName in this.Switches) {
                        // copy values
                        this.$set(this.Switches, switchName, updatedDevice);
                    }
                }
            }.bind(this));

            this.ws.addEventListener('open', function () {
                this.connected = true
                setTimeout(function () {
                    this.hideConnectionMsg = true;
                }.bind(this), 1500)
            }.bind(this));

            this.ws.addEventListener('close', function () {
                this.connected = false
                this.hideConnectionMsg = false;
                for (sw in this.Switches) {
                    this.removeSwitch(this.Switches[sw]);
                }
                this.Switches = {}
            }.bind(this));
        },

        // send a request to the server to set a particular port
        setPort: function (switchName, portName, terminalName, terminalState) {

            this.$http.put("/api/switch/" + switchName + "/port",
                JSON.stringify({
                    name: portName,
                    terminals: [{ "name": terminalName, "state": terminalState }],
                }));
        },
        setPorts: function(switchName, portName, terminals) {
            this.$http.put("/api/switch/" + switchName + "/port",
            JSON.stringify({
                name: portName,
                terminals: terminals,
            }));
        },
    },

    computed: {

        loading: function () {
            if (Object.keys(this.Switches).length == 0) {
                return false;
            }
            return true;
        }
    }
});