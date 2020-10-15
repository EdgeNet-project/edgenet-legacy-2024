
const theme = {
    global: {
        font: {
            family: '"Source Sans Pro", sans-serif',
        },
        colors: {
            icon: {
                0: "#",
                1: "6",
                2: "6",
                3: "6",
                4: "6",
                5: "6",
                6: "6",
                dark: "#f8f8f8",
                light: "#666666"
            },
            active: "rgba(221,221,221,0.5)",
            black: "#000000",
            border: {
                dark: "rgba(255,255,255,0.33)",
                light: "rgba(0,0,0,0.33)"
            },
            brand: "#5785DC",
            control: {
                dark: "accent-1",
                light: "brand"
            },
            focus: "#6FFFB0",
            placeholder: "#AAAAAA",
            selected: "brand",
            text: {
                dark: "#f8f8f8",
                light: "#444444"
            },
            white: "#FFFFFF",
            "accent-1": "#9c2846",
            "accent-2": "#d75f76",
            "accent-3": "accent-2",
            "accent-4": "accent-2",
            "dark-1": "#424656",
            "dark-2": "#a6abbd",
            "dark-3": "dark-2",
            "dark-4": "dark-2",
            "dark-5": "dark-2",
            "dark-6": "dark-2",
            "light-1": "#F8F8F8",
            "light-2": "#F2F2F2",
            "light-3": "#EDEDED",
            "light-4": "#DADADA",
            "light-5": "#DADADA",
            "light-6": "#DADADA",
            "neutral-1": "#5785dc",
            "neutral-2": "#8290bb",
            "neutral-3": "#f4f9ff",
            "neutral-4": "#e6f4f1",
            "status-critical": "#9c2846",
            "status-error": "#9c2846",
            "status-warning": "#d5a419",
            "status-ok": "#008a62",
            "status-unknown": "#a6abbd",
            "status-disabled": "#a6abbd"
        },
        input: {
            weight: 300,
        },
    },
    anchor: {
        fontWeight: 300,
        color: {
            dark: "accent-1",
            light: "brand"
        },
        hover: {
            // textDecoration: 'none'
        }
    },
    button: {
        default: {
            border: {
                radius: "4px",
                color: {
                    light: "brand"
                },
            },
        },

        // color: {
        //     light: "#F8F8F8"
        // },

        hover: {
            border: {
                radius: "4px",
                color: {
                    light: "brand"
                },
                width:"1px",
            }
        //     background: {
        //         color: {
        //             light: "dark-1"
        //         },
        //     },
        },
        primary: {
            color: {
                light: "#F8F8F8"
            },
            border: {
                radius: "4px",
                color: {
                    light: "black"
                },
                width:"1px"
            },
            background: {
                color: {
                    light: "brand"
                },
            },
        //     border: {
        //         color: {
        //             light: "brand"
        //         },
        //     }
        },
        secondary: {
            border: {
                radius: "4px",
                color: {
                    light: "brand"
                },
            },
        },
        // default: {
        //     // border: {
        //     //     radius: "4px",
        //     //     color: {
        //     //         light: "brand"
        //     //     },
        //     // },
        //     background: {
        //         color: {
        //             // light: "brand"
        //         },
        //     },
        // },

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