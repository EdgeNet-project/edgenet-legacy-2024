import React, { useContext } from 'react';
import { AuthenticationContext } from "../AuthenticationContext";
import AUP from "../views/AUP";

const Authenticated = ({children}) => {
    const { isAuthenticated, edgenet } = useContext(AuthenticationContext);


    if (!isAuthenticated()) {
        return null;
    }

    if (!edgenet) {
        return 'edgenet'
    }

    if (!edgenet.status.aup) {
        return <AUP />
    }

    return children;
};

export default Authenticated;
