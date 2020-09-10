import React from "react";
import { Box } from "grommet";

const AuthorityAddress = ({resource}) =>
    <Box>
        {resource.spec.address.street} <br />
        {resource.spec.address.zip} {resource.spec.address.city} <br />
        {resource.spec.address.region} {resource.spec.address.country}
    </Box>;

export default AuthorityAddress;