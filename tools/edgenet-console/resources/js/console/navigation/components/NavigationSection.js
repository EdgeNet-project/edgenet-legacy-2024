import React from "react";
import {Box, Nav, Text} from "grommet";

const NavigationSection = ({children, label}) =>
    <Nav gap="none">
        {label && <Box border={{side:'top',color:'light-4'}} pad={{horizontal:'medium', vertical:'small'}} margin={{top:'small'}}>
            <Text size="small">{label}</Text>
        </Box>}
        {children}
    </Nav>;

export default NavigationSection;