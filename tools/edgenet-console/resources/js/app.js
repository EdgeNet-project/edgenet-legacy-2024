import React from "react";
import ReactDOM from "react-dom";
import { Console } from "./console";
import { Server, ServerCluster, Chat, Organization, User } from "grommet-icons";

const settings = {
    logo: "/images/edgenet.png",

    api: {
        server: null, // localhost
        prefix: ''

    },

    navigation: [
        {
            menu: [
                {
                    name: 'nodes',
                    label: 'Nodes',
                    path: '/nodes',
                    icon: <Server />,
                },
            ]
        },
        {
            label: 'My Authority',
            menu: [
                {
                    name: 'slices',
                    label: 'Slices',
                    path: '/slices',
                    icon: <ServerCluster />,
                },
                {
                    name: 'users',
                    label: 'Users',
                    path: '/users',
                    icon: <User />,
                },
                {
                    name: 'authority',
                    label: 'My Authority',
                    path: '/authority',
                    icon: <Organization />,
                },
            ]
        },
        {
            label: 'Requests',
            menu: [
                {
                    name: 'userrequests',
                    label: 'Users',
                    path: '/userrequests',
                    icon: <User />,
                },
                {
                    name: 'authorityrequests',
                    label: 'Authorities',
                    path: '/authorityrequests',
                    icon: <Organization />,
                },
            ]
        }


    ]
};

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

const dom = document.getElementById('console');
if (dom) {
    ReactDOM.render(<Console settings={settings} />, dom);
}
