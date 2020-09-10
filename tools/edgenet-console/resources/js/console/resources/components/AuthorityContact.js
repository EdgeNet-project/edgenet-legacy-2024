import React from "react";
import { Box, Anchor } from "grommet";

const AuthorityContact = ({resource}) =>
    <Box>
        {resource.spec.contact.firstname} {resource.spec.contact.lastname} <br />
        <Anchor href={"mailto:" + resource.spec.contact.email}>{resource.spec.contact.email}</Anchor> <br />
        {resource.spec.contact.phone}
    </Box>;

export default AuthorityContact;