var Switch = {
    components: {
        'swbutton': Button,
        'device-name': DeviceName,
    },
    template: `
        <div class="switch">
            <device-name :name="name"></device-name>
            <div v-for="port in ports">
            <div class="port"> Port {{port.name}}
                <div class="btn-group" role="group" aria-label="..." v-for="terminal in port.terminals">
                <swbutton :label="terminal.name" :port="port.name" :state="terminal.state" v-on:set-port="setPort" v-on:set-port-exclusive="setPortExclusive">
                </swbutton>
                </div>
            </div>
            </div>
        </div>`,
    props: {
        name: String,
        ports: Array,
    },
    mounted: function () { },
    beforeDestroy: function () { },
    methods: {
        // set one terminal of a port
        setPort: function (portName, terminalName, terminalState) {

            this.$emit("set-port", this.name, portName, terminalName, terminalState)
        },
        // disable all terminals, except of the selected one
        setPortExclusive: function (portName, terminalName) {
            var port = undefined;
            var terminals = [];

            for (var i = 0; i < this.ports.length; i++) {
                if (this.ports[i].name == portName) {
                    port = this.ports[i]
                }
            }

            for (var i = 0; i < port.terminals.length; i++) {
                var terminal = {
                    name: port.terminals[i].name,
                    state: false,
                }
                if (terminal.name == terminalName){
                    terminal.state = true
                }
                terminals.push(terminal)
            }
            this.$emit("set-ports", this.name, portName, terminals)

        },
    },
    watch: {},
}