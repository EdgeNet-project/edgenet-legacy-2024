import React, { useContext } from 'react';
import { AuthenticationContext } from "../AuthenticationContext";

const Admin = ({children}) => {
    const { isAdmin } = useContext(AuthenticationContext);

    if (!isAdmin()) {
        return null;
    }

    return children;
};

export default Admin;
