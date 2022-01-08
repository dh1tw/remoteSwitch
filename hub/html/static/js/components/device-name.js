var DeviceName = {
    template: `
        <div class="device-name">
            <div class="tag col-auto">
                {{typeLabel}}
            </div>
            <div class="name col-auto">
                {{name}}
            </div>
        </div>`,
    props: {
        name: String,
        width: Number,
    },
    mounted: function () { },
    beforeDestroy: function () { },
    methods: {},
    computed: {
        typeLabel: function () {
            return "SW"
        },
    },
    watch: {},
}