import React from 'react';
import { UserConsumer } from "../UserContext";

const Authenticated = ({children}) =>
    <UserConsumer>
        { ({user}) => user.token ? children : null}
    </UserConsumer>;

export default Authenticated;
