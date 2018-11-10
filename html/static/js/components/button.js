var Button = {
    template: `<button class="btn sw-button" type="button" v-bind:class="{'btn-success':active, 'btn-primary':inactive}" v-on:click="setPort()">
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
            this.$emit("set-port", this.port, this.label)
        },
    },
    watch: {},
    computed: {
        inactive: function(){
            return !this.active;
        }
    }
}