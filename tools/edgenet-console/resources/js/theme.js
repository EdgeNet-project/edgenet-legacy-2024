
const theme = {
    global: {
        font: {
            family: '"Source Sans Pro", sans-serif',
        },
        colors: {
            brand: '#135ba5'
        },
        input: {
            weight: 300,
        },
    },
    anchor: {
        fontWeight: 300,
        color: {
            "dark": "accent-1",
            "light": "brand"
        },
        hover: {
            // textDecoration: 'none'
        }
    },
    button: {
        border: {
            radius: "8px",
        },

    },

    formField: {
        label: {
            color: "dark-2",
            size: "small",
            margin: {vertical: "0", bottom:"xsmall", horizontal: "0"},
            weight: 300
        },
        border: {
            color: "brand",
            side: "all",
            round: "4px"
        }
    }
};

export default theme;