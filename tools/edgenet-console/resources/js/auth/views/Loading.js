import React from "react";
import {Box} from "grommet";
import Header from "./Header";


const Loading = ({title}) =>
    <Box align="center">
        <Header title={title} />
        <Box pad={{vertical:'medium'}}>
        Please wait...
        </Box>
    </Box>

export default Loading;