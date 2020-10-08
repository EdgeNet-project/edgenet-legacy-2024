import React, { useContext } from 'react';
import { AuthenticationContext } from "../AuthenticationContext";

const Admin = ({children}) => {
    const { isAdmin, isClusterAdmin } = useContext(AuthenticationContext);

    if (!isAdmin() && !isClusterAdmin()) {
        return null;
    }

    return children;
};

export default Admin;
