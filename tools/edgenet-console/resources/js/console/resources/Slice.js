import React from "react";
import { Box, Text } from "grommet";

const Slice = ({resource}) =>
    <Box pad="small">
        <Text size="small">
            {resource.metadata.uid}
        </Text>
        {resource.metadata.name}
    </Box>;

export { Slice };
