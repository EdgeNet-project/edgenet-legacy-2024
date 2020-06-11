import React from 'react';
import { Box, Heading } from "grommet";
import { DataSourceConsumer } from "../DataSource";

const FormHeader = ({children, label}) =>
    <DataSourceConsumer>
        {
            ({item, id}) => item &&
                <Box pad={{bottom:'small'}}>
                    <Heading margin="none" level="2">{id ? children && children(item) : label}</Heading>
                </Box>
        }
    </DataSourceConsumer>;

export default FormHeader;