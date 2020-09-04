import React from "react";
import {Link} from "react-router-dom";
import {Box} from "grommet";

const Footer = () =>
    <Box direction="row" >
        <Link to="/">Go back to the login page</Link>
    </Box>;

export default Footer;