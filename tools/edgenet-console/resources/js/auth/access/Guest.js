import React from 'react';
import { AuthConsumer } from "../AuthContext";

const Guest = ({children}) =>
    <AuthConsumer>
        { ({isGuest}) => isGuest() ? children : null}
    </AuthConsumer>;

export default Guest;
