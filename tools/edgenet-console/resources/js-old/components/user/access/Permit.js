import React from 'react';
import { UserConsumer } from "../UserContext";

const Permit = ({children, ...props}) =>
    <UserConsumer>
        { ({user}) => Object.keys(props).some((p) => user[p]) ? children : null}
    </UserConsumer>;

export default Permit;
