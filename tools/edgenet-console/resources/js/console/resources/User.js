import React from "react";
import { Box, Anchor } from "grommet";

const User = ({resource}) =>
    <Box>
        {resource.spec.firstname} {resource.spec.lastname} <br />
        <Anchor href={"mailto:" + resource.spec.email}>{resource.spec.email}</Anchor> <br />
        {resource.spec.phone}
    </Box>;

export { User };