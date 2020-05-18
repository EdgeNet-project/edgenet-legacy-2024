import React from "react";
import { Box } from "grommet";

const NewsView = ({item}) =>
    <Box pad="medium">
        {item.text}
    </Box>;

export default NewsView;
