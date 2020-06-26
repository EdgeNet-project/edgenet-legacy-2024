import React from "react";
import {Box, Image} from "grommet";

const Header = ({title, logo = '/images/edgenet.png'}) =>
    <Box gap="small" pad={{vertical:'small'}} align="center">
        {logo && <Image style={{maxWidth:'25%',margin:'0 auto'}} src={logo} alt={title} />}
        {title ? title : "Signup"}
    </Box>;

export default Header;