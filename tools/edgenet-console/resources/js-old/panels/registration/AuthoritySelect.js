import React from "react";

import {Anchor, Box, Text, Button} from "grommet";
import {DataSource} from "../../components/datasource";
import {List} from "../../components/datasource/list";

const AuthorityListElement = ({item}) =>
    <Box pad={{ vertical: "small" }}>
        {item.fullname}
        <Text size="small">
            {item.address}
        </Text>
        <Text size="small">
            {item.shortname} - <a target="_blank" href={item.url}>{item.url}</a>
        </Text>
    </Box>;

const AuthoritySelect = ({selected, onSelect, setStep}) =>
    <Box>
        <Box pad={{vertical:"medium"}}>
            <Text color="dark-2">Please select the institution you are part of</Text>
        </Box>
        <Box width="medium" height="small">
            <DataSource source="/sites" currentId={selected}>
                <List onRowClick={onSelect}>
                    <AuthorityListElement />
                </List>
            </DataSource>
        </Box>
        <Box pad={{vertical:"medium"}} direction="row" justify="end" align="center">
            <Box pad={{right:"small"}} margin={{right:"small"}}>
                <Anchor alignSelf="start" label="My institution is not on the list" onClick={() => setStep(1)} />
            </Box>
            <Button primary label="Continue" onClick={null} />
        </Box>
    </Box>;

export default AuthoritySelect;