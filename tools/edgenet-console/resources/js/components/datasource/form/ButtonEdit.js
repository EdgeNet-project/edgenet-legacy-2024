import React from "react";
import LocalizedStrings from "react-localization";
import { Box, Button } from "grommet";
import { Edit } from "grommet-icons";
import { DataSourceConsumer } from "../DataSource";

const strings = new LocalizedStrings({
    en: {
        edit: "Edit"
    },
    fr: {
        edit: "Modifier"
    }
});

const ButtonEdit = ({id}) =>
    <DataSourceConsumer>
        {
            ({ getItem }) =>
                <Button icon={<Edit />} onClick={() => getItem(id)} label={strings.edit} />
        }
    </DataSourceConsumer>;

export default ButtonEdit;