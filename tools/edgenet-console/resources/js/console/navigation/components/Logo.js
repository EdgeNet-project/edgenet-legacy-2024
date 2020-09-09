import React, {useContext} from "react";
import {Box, Image, Text} from "grommet";

import {ConsoleContext} from "../../index";

const Logo = () => {
    const { logo, title } = useContext(ConsoleContext);

    return (
        <Box pad="medium" align="start">
            {logo ? <Image fill src={logo} alt={title} /> : <Text>{title}</Text>}
        </Box>
    );
};

export default Logo;