import React from "react";

import { Box } from "grommet";
import { DataConsumer } from "../Data";

const Info = ({label = "items"}) =>
    <Box pad="small" flex="grow" direction="row">
        <DataConsumer>
            {({total, count, loading}) => {
                if (loading) return '...';
                return ((total > count) ? count + ' of ' + total : total) + ' ' + label;
            }}
        </DataConsumer>
    </Box>;

export default Info;
