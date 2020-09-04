import React, {useContext} from "react";
import {Box, Image, Text} from "grommet";
import {AuthenticationContext} from "../AuthenticationContext";

const Logo = () => {
    const { title, logo, prefix } = useContext(AuthenticationContext);

    return (
        <Box pad={{vertical:'large', right:'xlarge'}} align="start">
            {logo ? <Image fill src={logo} alt={title || document.title} /> : <Text>{title || document.title}</Text>}
        </Box>
    );
};

export default Logo;
