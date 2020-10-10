import React, { useContext } from 'react';
import { AuthenticationContext } from "../AuthenticationContext";

const ClusterAdmin = ({children}) => {
    const { isClusterAdmin } = useContext(AuthenticationContext);

    if (!isClusterAdmin()) {
        return null;
    }

    return children;
};

export default ClusterAdmin;
