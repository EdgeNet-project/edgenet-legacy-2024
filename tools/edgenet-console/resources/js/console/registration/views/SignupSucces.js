import React from "react";
import {Box} from "grommet";
import {Link} from "react-router-dom";
import { ConsoleLogo } from "../../index";

const SignupSucces = () =>
    <Box align="center">
        <ConsoleLogo />
        <Box pad={{vertical:'medium'}}>
            Thank you for signin up!<br/>
            You will receive shortly an email asking to validate your email address.<br/>
            Once validate we will review your information and come back to you!
        </Box>
        <Box direction="row" >
            <Link to="/">Go back to the login page</Link>
        </Box>
    </Box>;

export default SignupSucces;