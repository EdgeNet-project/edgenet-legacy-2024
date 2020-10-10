import React from "react";
import { Box, Text } from "grommet";

const NodeList = ({resource}) =>
    <Box pad="small">
        <Text size="small">
            {resource.metadata.uid}
        </Text>
        {resource.metadata.name}
    </Box>;

export default NodeList;
