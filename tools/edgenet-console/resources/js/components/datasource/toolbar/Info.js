import React from "react";

import { Box } from "grommet";
import { DataSourceConsumer } from "../DataSource";

const Info = ({label = "items"}) =>
    <Box pad="small" flex="grow" direction="row">
        <DataSourceConsumer>
            {({total, count, loading}) => {
                if (loading) return '...';
                return ((total > count) ? count + ' of ' + total : total) + ' ' + label;
            }}
        </DataSourceConsumer>
    </Box>;

export default Info;