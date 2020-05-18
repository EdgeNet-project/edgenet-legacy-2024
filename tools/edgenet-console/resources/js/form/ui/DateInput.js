import React from "react";
import moment from "moment";

import { Box, Text, Button, Calendar, DropButton, FormField } from "grommet";
import { Schedule } from "grommet-icons";

const DateInput = ({value, label, onChange}) => {
    const [date, setDate] = React.useState();
    React.useEffect(() => {
        setDate(value);
    }, []);
    return (
        <DateDropButton
            label={label} value={date}
            onChange={(date) => { setDate(date); onChange({target: {value: date}})}}
        />
    );
};


const FormFieldDate = ({name, label}) =>
    <FormField name={name} label={label} plain={false} component={DateInput} />;


const DropContent = ({ date, onClose, onClear }) => {
    return (
        <Box align="center">
            <Calendar
                locale="fr-FR"
                animate={false}
                date={date ? moment(date, "DD/MM/YYYY").format("MM/DD/YYYY") : undefined}
                onSelect={onClose}
                showAdjacentDays={false}
            />
            <Box flex={false} pad="small">
                <Button label="Effacer" onClick={onClear} />
            </Box>
        </Box>
    );
};

const DateDropButton = ({label, value, onChange}) => {
    const [open, setOpen] = React.useState();

    const onClose = (nextDate) => {
        onChange && onChange(moment(nextDate).format("DD/MM/YYYY"));
        setOpen(false);
        setTimeout(() => setOpen(undefined), 1);
    };

    const onClear = () => {
        onChange && onChange(null);
        setOpen(false);
    };
    return (
        <DropButton
            open={open}
            onClose={() => setOpen(false)}
            onOpen={() => setOpen(true)}
            dropContent={
                <DropContent date={value} onClose={onClose} onClear={onClear} />
            }
        >
            <Box>
                <Box direction="row" pad="small" gap="small" justify="around" background="white"
                     border={{ color: 'dark-2' }} width="small">
                    <Text color={value ? undefined : "dark-5"}>
                        {value ? value : label}
                    </Text>
                    <Schedule />
                </Box>
            </Box>
        </DropButton>
    );
};

export default FormFieldDate;
