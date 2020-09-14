import React, { useContext } from 'react';
import { AuthenticationContext } from "../AuthenticationContext";
import AUP from "../views/AUP";

const Authenticated = ({children}) => {
    const { isAuthenticated, aup, edgenet, loading } = useContext(AuthenticationContext);

    if (!isAuthenticated()) {
        return null;
    }

    if (loading) {
        return null;
    }

    if (!edgenet) {
        return 'edgenet error'
    }

    if (aup && !aup.accepted) {
        return <AUP />
    }

    return children;
};

export default Authenticated;
