var DeviceName = {
    template: '<div class="device-name" :style="styleObj"><div class="tag">{{typeLabel}}</div><div class="name">{{name}}</div></div>',
    props: {
        name: String,
        width: Number,
    },
    mounted: function () {},
    beforeDestroy: function () {},
    methods: {},
    computed: {
        typeLabel: function () {
            return "SW"
        },
        styleObj: function() {
            return {
                "max-width": this.width + 'px',
                "font-size": this.width / 15 + "pt",
            }
        }
    },
    watch: {},
}