import React, { useState } from "react";
import axios from "axios";

const EdgenetContext = React.createContext({});

const Edgenet = ({children}) => {
    const [ user, setUser ] = useState();
    const [ resources, setResources ] = useState();

    const api = document.querySelector('meta[name="k8s-api-server"]')?.getAttribute('content');
    if (!api) {
        throw ('API endpoint configuration not found: a meta tag k8s-api-server with the K8s API server address and port should exist');
    }

    const getUser = function(authority, user) {
        axios.get(api + '/apis/apps.edgenet.io/v1alpha/namespaces/authority-'+authority+'/users/'+user)
            .then(({data}) => console.log(data));
    }

    const getResources = function(url) {
        // const { items, current_page, last_page, queryParams } = this.state;

        // if (!api) return false;

        // if (current_page >= last_page) return;

        //const url = '';
        axios.get(api + url, {
            // params: { ...queryParams, page: current_page + 1 },
            // paramsSerializer: qs.stringify,
        })
            .then(({data}) => {
                setResources(data.items);
                // this.setState({
                //     ...data, loading: false
                // });
            })
            .catch(error => {
                console.log(error)
            });
    }

    return (
        <EdgenetContext.Provider value={{
            api: api,
            user: user,

            resources: resources,
            getResources: getResources

        }}>
            {children}
        </EdgenetContext.Provider>
    );
}

export { Edgenet, EdgenetContext }