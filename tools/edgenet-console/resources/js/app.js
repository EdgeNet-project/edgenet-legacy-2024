import React from "react";
import ReactDOM from "react-dom";
import { Console, Routes } from "./core";
import { Server } from "grommet-icons";

import theme from "./theme";


const menu = [
    {
        label: 'Nodes',
        path: '/nodes',
        icon: <Server />,
        resource: 'nodes'
    },

];

const resources = [
        {
            name: 'nodes',
            api: {
                type: 'k8s',
                url: '/api/v1/nodes',
                server: 'http://192.168.10.8'
            },
            media: [],
        },

    ]

const config = {
    resources: resources,

};


const dom = document.getElementById('application');
if (dom) {
    ReactDOM.render(
        <Console resources={resources}>
            <Routes menu={menu} theme={theme} />
        </Console>,
        dom
    );
}
