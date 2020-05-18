import React from "react";
import { StatusGood, StatusDisabled } from "grommet-icons";

import SelectFilter from "./SelectFilter";

const defaultOptions = [
    {
        label: 'Enabled',
        value: 1,
        icon: <StatusGood color="status-ok" />

    },
    {
        label: 'Disabled',
        value: 0,
        icon: <StatusDisabled color="status-disabled" />
    }
];

const EnabledFilter = ({label="Enabled filter", name="enabled", options=defaultOptions}) =>
    <SelectFilter label={label} name={name} options={options} />;

export default EnabledFilter;
