var Button = {
    template: `<button class="btn sw-button" type="button" v-bind:class="{'btn-success':state, 'btn-primary':inverted_state}" v-on:click="setPort()">
                    {{label}}
                </button>`,
    props: {
        label: String,
        state: Boolean,
        port: String,
    },
    mounted: function(){},
    beforeDestroy: function(){},
    methods: {
        setPort: function(){
            this.$emit("set-port", this.port, this.label, !this.state)
        },
    },
    watch: {},
    computed: {
        inverted_state: function(){
            return !this.state;
        }
    }
}