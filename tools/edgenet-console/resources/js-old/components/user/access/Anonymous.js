import React from 'react';
import { UserConsumer } from "../UserContext";

const Anonymous = ({children}) =>
    <UserConsumer>
        { ({user}) => !user.token ? children : null}
    </UserConsumer>;

export default Anonymous;
