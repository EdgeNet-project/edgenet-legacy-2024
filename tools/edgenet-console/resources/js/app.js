import React from "react";
import ReactDOM from "react-dom";
import { Console, Routes } from "./core";
import { Edgenet } from "./edgenet";
import { Server, ServerCluster, Chat, Organization, User } from "grommet-icons";

import theme from "./theme";


const menu = [
    {
        label: 'Nodes',
        path: '/nodes',
        main: true,
        icon: <Server />,
        resource: 'nodes'
    },
    {
        label: 'Slices',
        path: '/slices',
        main: true,
        icon: <ServerCluster />,
        resource: 'nodes'
    },
    {
        label: 'User Requests',
        path: '/userregistrationrequests',
        main: false,
        icon: <Chat />,
        resource: 'userregistrationrequests'
    },
    {
        label: 'Authority',
        path: '/requests',
        main: false,
        icon: <Organization />,
        resource: 'nodes'
    },


];

const resources = [
    {
        name: 'nodes',
        type: 'k8s',
        api: '/api/v1/nodes',
        media: [],
    },
    {
        name: 'userregistrationrequests',
        api: '/apis/apps.edgenet.io/v1alpha/namespaces/authority-cslash/userregistrationrequests',
        type: 'k8s',
        media: [],
    },


    ]

const config = {
    resources: resources,

};


const dom = document.getElementById('application');
if (dom) {
    ReactDOM.render(
        <Edgenet>
            <Console resources={resources}>
                <Routes menu={menu} theme={theme} />
            </Console>
        </Edgenet>,
        dom
    );
}
