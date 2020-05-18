import React from 'react';
import { AuthConsumer } from "../AuthContext";

const Authenticated = ({children}) =>
    <AuthConsumer>
        { ({isAuthenticated}) => isAuthenticated() ? children : null}
    </AuthConsumer>;

export default Authenticated;
