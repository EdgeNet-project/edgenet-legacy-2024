import React, { useState } from "react";
import axios from "axios";

const EdgenetContext = React.createContext({});

const Edgenet = ({children}) => {
    const [ user, setUser ] = useState();

    const api = document.querySelector('meta[name="k8s-api-server"]')?.getAttribute('content');
    if (!api) {
        throw ('API endpoint configuration not found: a meta tag k8s-api-server with the K8s API server address and port should exist');
    }

    const getUser = function() {
        axios.get(api + '/apis/apps.edgenet.io/v1alpha/namespaces/authority-cslash/users/ciro')
            .then(({data}) => console.log(data));
    }

    return (
        <EdgenetContext.Provider value={{
            api: api,
            user: user,

        }}>
            {children}
        </EdgenetContext.Provider>
    );
}

export { Edgenet, EdgenetContext }