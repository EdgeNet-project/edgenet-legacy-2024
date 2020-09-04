import React from "react";
import { Box, Text } from "grommet";

const List = ({item, resource, onClick}) =>
    <Box pad="small" onClick={() => onClick(item.id)}>
        <Text size="small">
            {item.metadata.uid}
        </Text>
        {item.metadata.name}
    </Box>;

export default List;
