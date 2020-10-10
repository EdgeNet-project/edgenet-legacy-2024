import React from "react";
import ReactDOM from "react-dom";
import { Console } from "./console";

const settings = {
    logo: "/images/edgenet.png",

    api: {
        server: null, // localhost
        prefix: ''

    },

};

const dom = document.getElementById('console');
if (dom) {
    ReactDOM.render(<Console settings={settings} />, dom);
}
