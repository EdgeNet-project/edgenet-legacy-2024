import React from "react";
import LocalizedStrings from "react-localization";
import { Button } from "grommet";
import { Refresh } from "grommet-icons";
import { DataSourceConsumer } from "../DataSource";

const strings = new LocalizedStrings({
    en: {
        reset: "Reset",
    },
    fr: {
        reset: "Réinitialiser",
    }
});

const ButtonDelete = () =>
    <DataSourceConsumer>
        {
            ({changed}) =>
                <Button icon={<Refresh />}
                        disabled={!changed}
                        type="reset"
                        label={strings.reset} />
        }
    </DataSourceConsumer>;

export default ButtonDelete;