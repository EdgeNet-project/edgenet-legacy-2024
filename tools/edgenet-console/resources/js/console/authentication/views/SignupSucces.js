import React from "react";
import {Box} from "grommet";
import Header from "./Header";
import Footer from "./Footer";

const SignupSucces = () =>
    <Box align="center">
        <Header />
        <Box pad={{vertical:'medium'}}>
            Thank you for signin up!<br/>
            You will receive shortly an email asking to validate your email address.<br/>
            Once validate we will review your information and come back to you!
        </Box>
        <Footer />
    </Box>;

export default SignupSucces;