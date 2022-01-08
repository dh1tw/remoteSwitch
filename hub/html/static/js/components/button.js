var Button = {
    template: `<button class="btn sw-button" v-bind:class="{'btn-success':state, 'btn-primary':inverted_state}" v-on:click="setPort()" @contextmenu="clickHandler($event)">
                    {{label}}
                </button>`,
    props: {
        label: String,
        state: Boolean,
        port: String,
    },
    mounted: function () { },
    beforeDestroy: function () { },
    methods: {
        // handle right clicks
        clickHandler: function (e) {
            this.$emit("set-terminal-exclusive", this.port, this.label);
            // omit standard right click menu
            e.preventDefault();
        },
        setPort: function () {
            this.$emit("set-terminal", this.port, this.label, !this.state);
        },
    },
    watch: {},
    computed: {
        inverted_state: function () {
            return !this.state;
        }
    }
};