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
                <swbutton :label="terminal.name" :port="port.name" :state="terminal.state" v-on:set-port="setPort">
                </swbutton>
                </div>
            </div>
            </div>
        </div>`,
    props: {
        name: String,
        ports: Array,
    },
    mounted: function () {},
    beforeDestroy: function () {},
    methods: {
        setPort: function(portName, terminalName) {
            this.$emit("set-port", this.name, portName, terminalName)
        },
    },
    watch: {},
}