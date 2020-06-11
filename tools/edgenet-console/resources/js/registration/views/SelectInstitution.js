import React from "react";

import {Box, Text} from "grommet";
import {DataSource} from "../../components/datasource";
import {List} from "../../components/datasource/list";
import Institution from "./Institution";

const SelectInstitution = ({selected, onSelect}) =>
    <Box>
        <Box pad={{vertical:"medium"}}>
            <Text color="dark-2">Please select the institution you are part of</Text>
        </Box>
        <Box width="medium" height="small">
            <DataSource source="/sites" currentId={selected}>
                <List onRowClick={onSelect}>
                    <Institution />
                </List>
            </DataSource>
        </Box>
    </Box>;

export default SelectInstitution;