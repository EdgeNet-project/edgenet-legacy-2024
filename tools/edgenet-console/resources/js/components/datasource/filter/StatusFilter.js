import React from "react";
import { StatusGood, StatusWarning, StatusCritical, StatusDisabled, StatusUnknown} from "grommet-icons";
import SelectFilter from "./SelectFilter";

const defaultOptions = [
    {
        label: 'Good',
        value: 0,
        icon: <StatusGood color="status-ok" />
    },
    {
        label: 'Warning',
        value: 1,
        icon: <StatusWarning color="status-warning" />
    },
    {
        label: 'Critical',
        value: 2,
        icon: <StatusCritical color="status-critical" />
    },
    {
        label: 'Disabled',
        value: 3,
        icon: <StatusDisabled color="status-disabled" />
    },
    {
        label: 'Unknown',
        value: 4,
        icon: <StatusUnknown color="status-unknown" />
    },
];

const StatusFilter = ({label="Status filter", name="status", options=defaultOptions, multi=true}) =>
    <SelectFilter label={label} name={name} options={options} multi={multi} />;


export default StatusFilter;
