import React from "react";
import LocalizedStrings from "react-localization";
import { Button, Text } from "grommet";
import { Save } from "grommet-icons";
import { DataSourceConsumer } from "../DataSource";

const strings = new LocalizedStrings({
    en: {
        save: "Save"
    },
    fr: {
        save: "Sauvegarder"
    }
});

const ButtonSave = ({label}) =>
    <DataSourceConsumer>
        {
            ({ itemChanged }) =>
                    <Button primary icon={<Save />}
                            disabled={!itemChanged}
                            type="submit"
                            label={<Text color="white">
                                <strong>{label ? label : strings.save}</strong>
                            </Text>} />
        }
    </DataSourceConsumer>;

export default ButtonSave;