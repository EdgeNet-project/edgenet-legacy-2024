import React from 'react';
import {Box} from "grommet";

import Header from "./Header";
import Footer from "./Footer";

const UserNotActive = () =>
        <Box align="center">
            <Header title="EdgeNet User not active" />
            <Box margin={{vertical:'medium'}} width="large" height="60vh">
                Your user hasn't been approved yet or it is not active.<br />
                Please contact your local Admin.
            </Box>
            <Footer />
        </Box>;

export default UserNotActive;