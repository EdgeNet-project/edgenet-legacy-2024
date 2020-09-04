import React, {useContext} from 'react';
import { AuthenticationContext } from "../AuthenticationContext";

const Guest = ({children}) => {
    const { isGuest } = useContext(AuthenticationContext);

    return isGuest() ? children : null;
}

export default Guest;
