import React from "react";
import { Box, Text, Anchor } from "grommet";
import { Link } from "grommet-icons";

const Authority = ({resource}) =>
    <Box>
        <Box direction="row" gap="xsmall">
            <Text>{resource.spec.fullname}</Text>
            <Anchor plain label=" " target="_blank" href={resource.spec.url} icon={<Link size="small" />} />
        </Box>
        <Text size="small">({resource.spec.shortname})</Text>
    </Box>;

const AuthorityContact = ({resource}) =>
    <Box>
        {resource.spec.contact.firstname} {resource.spec.contact.lastname} <br />
        <Anchor href={"mailto:" + resource.spec.contact.email}>{resource.spec.contact.email}</Anchor> <br />
        {resource.spec.contact.phone}
    </Box>;

const AuthorityAddress = ({resource}) =>
    <Box>
        {resource.spec.address.street} <br />
        {resource.spec.address.zip} {resource.spec.address.city} <br />
        {resource.spec.address.region} {resource.spec.address.country}
    </Box>;

export { Authority, AuthorityAddress, AuthorityContact }