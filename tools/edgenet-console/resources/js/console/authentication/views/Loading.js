import React from "react";
import {Box} from "grommet";
import { ConsoleLogo } from "../../index";


const Loading = () =>
    <Box gap="medium" alignSelf="center" width="medium" alignContent="center" align="stretch">
        <Box margin={{vertical:'medium'}}>
            <ConsoleLogo />
        </Box>
        <Box pad={{vertical:'medium'}}>
            Please wait...
        </Box>
    </Box>

export default Loading;