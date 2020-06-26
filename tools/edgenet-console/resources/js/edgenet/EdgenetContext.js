import React from "react";

const EdgenetContext = React.createContext({});

const Edgenet = ({children}) => {

    const api = document.querySelector('meta[name="k8s-api-server"]')?.getAttribute('content');
    if (!api) {
        throw ('API endpoint configuration not found: a meta tag k8s-api-server with the K8s API server address and port should exist');
    }

    return (
        <EdgenetContext.Provider value={{
            api: api
        }}>
            {children}
        </EdgenetContext.Provider>
    );
}

export { Edgenet, EdgenetContext }