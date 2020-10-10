import React, { useContext } from 'react';
import { AuthenticationContext } from "../AuthenticationContext";
import AUP from "../views/AUP";
import UserNotActive from "../views/UserNotActive";

const Authenticated = ({children}) => {
    const { isAuthenticated, aup, edgenet, loading } = useContext(AuthenticationContext);

    if (!isAuthenticated()) {
        return null;
    }

    if (loading) {
        return null;
    }

    if (!edgenet) {
        return <UserNotActive />
    }

    if (aup && !aup.accepted) {
        return <AUP />
    }

    return children;
};

export default Authenticated;
