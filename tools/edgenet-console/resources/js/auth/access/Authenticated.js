import React, { useContext } from 'react';
import { AuthContext } from "../AuthContext";
import AUP from "../views/AUP";

const Authenticated = ({children}) => {
    const { isAuthenticated, edgenet } = useContext(AuthContext);


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
