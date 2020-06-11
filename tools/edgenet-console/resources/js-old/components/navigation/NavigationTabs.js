import React from "react";
import { Box } from "grommet";

const NavigationTabs = ({children}) =>
    <Box direction="row"
         justify="center"
         flex={false}
         border={{side:'bottom',size:'small',color:'brand'}}
         wrap>
        {children}
    </Box>;

export default NavigationTabs;
