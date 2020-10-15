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

export default Authority