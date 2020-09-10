import React from "react";
import { Box, Text, Anchor } from "grommet";

const Authority = ({resource}) =>
    <Box>
        {resource.spec.fullname} <Text size="small">({resource.spec.shortname})</Text> <br />
        <Text size="small">
            <Anchor target="_blank" href={resource.spec.url}>{resource.spec.url}</Anchor>
        </Text>
    </Box>;

export default Authority