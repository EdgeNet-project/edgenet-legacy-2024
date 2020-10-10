import React from 'react';
import { UserConsumer } from "../UserContext";

const Deny = ({children, ...props}) =>
    <UserConsumer>
        { ({user}) => Object.keys(props).some((p) => user.hasOwnProperty(p) && user[p] === true) ? null : children}
    </UserConsumer>;

export default Deny;
