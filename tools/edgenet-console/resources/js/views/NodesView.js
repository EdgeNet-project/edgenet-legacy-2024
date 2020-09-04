import React from "react";
import { Box } from "grommet";

const NodesView = ({item}) =>
    <Box pad="medium">
        {item.text}
    </Box>;

export default NodesView;
