var Switch = {
    components: {
        'swbutton': Button,
        'device-name': DeviceName,
    },
    template: `
        <div class="switch container-xl">
            <device-name :name="name"></device-name>
            <div class="row align-items-center flex-nowrap port" v-for="port in ports">
                <div class="name col-auto">
                    Port {{port.name}}
                </div>
                <div class="col-md-auto col-11">
                    <div class="row g-1">
                        <div class="col-auto" v-for="terminal in port.terminals">
                            <swbutton :label="terminal.name" :port="port.name" :state="terminal.state" v-on:set-terminal="setTerminal" v-on:set-terminal-exclusive="setTerminalExclusive">
                            </swbutton>
                        </div>
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
        setTerminal: function (portName, terminalName, terminalState) {

            this.$emit("set-terminal", this.name, portName, terminalName, terminalState);
        },
        // disable all terminals, except of the selected one
        setTerminalExclusive: function (portName, terminalName) {
            var port = undefined;
            var terminals = [];

            for (var i = 0; i < this.ports.length; i++) {
                if (this.ports[i].name == portName) {
                    port = this.ports[i];
                }
            }

            for (var i = 0; i < port.terminals.length; i++) {
                var terminal = {
                    name: port.terminals[i].name,
                    state: false,
                };
                if (terminal.name == terminalName) {
                    terminal.state = true;
                }
                terminals.push(terminal);
            }
            this.$emit("set-port", this.name, portName, terminals);
        },
    },
    watch: {},
}