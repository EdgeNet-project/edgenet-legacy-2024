import React from "react";
import LocalizedStrings from "react-localization";
import { Box, Button } from "grommet";
import { Close } from "grommet-icons";
import { DataSourceConsumer } from "../DataSource";

const strings = new LocalizedStrings({
    en: {
        cancel: "Cancel",
    },
    fr: {
        cancel: "Annuler",
    }
});

const ButtonCancel = () =>
    <DataSourceConsumer>
        {
            ({item, itemChanged, unsetItem}) => item &&
                <Box pad="small">
                    <Button plain icon={<Close />}
                            disabled={!itemChanged && !item.id}
                            label={strings.cancel}
                            onClick={unsetItem} />
                </Box>
        }
    </DataSourceConsumer>;

export default ButtonCancel;