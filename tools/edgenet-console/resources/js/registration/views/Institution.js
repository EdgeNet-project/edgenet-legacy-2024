import {Box, Text} from "grommet";
import React from "react";

const Institution = ({item}) =>
    <Box pad={{ vertical: "small" }}>
        {item.fullname}
        <Text size="small">
            {item.address}
        </Text>
        <Text size="small">
            {item.shortname} - <a target="_blank" href={item.url}>{item.url}</a>
        </Text>
    </Box>;

export default Institution;