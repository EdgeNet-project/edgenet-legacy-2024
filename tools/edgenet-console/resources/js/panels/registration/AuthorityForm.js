import React from "react";
import {Anchor, Box, Form, FormField, Text, TextInput, Button} from "grommet";

const AuthorityForm = ({setAuthority, setStep}) => {
    const [value, setValue] = React.useState({
        name: "",
        shortname: "",
        address: "",
        zipcode: "",
        city: "",
        country: "",
        url: ""
    });

    return (
        <Box width="medium">
            <Form value={value}
                  onChange={nextValue => setValue(nextValue)}
                  onReset={() => setValue({})}
                  onSubmit={({ value }) => setAuthority(value)}>
                <Box pad={{vertical: "medium"}}>
                    <Text color="dark-2">Please complete with the information of the institution you are part of</Text>
                </Box>
                <Box>
                    <FormField label="Institution full name" name="name" required />
                    <FormField label="Institution shortname or initials" name="shortname">
                        <TextInput name="shortname" required />
                    </FormField>
                    <FormField label="Address">
                        <TextInput name="address" required />
                    </FormField>
                    <FormField label="Zipcode">
                        <TextInput name="zipcode" required />
                    </FormField>
                    <FormField label="City">
                        <TextInput name="city" required />
                    </FormField>
                    <FormField label="Country">
                        <TextInput name="country" required />
                    </FormField>
                    <FormField label="Web page">
                        <TextInput name="url" required />
                    </FormField>

                </Box>
                <Box pad={{vertical:"medium"}} direction="row" justify="end" align="center">
                    <Box pad={{right:"small"}} margin={{right:"small"}}>
                        <Anchor alignSelf="start" label="Cancel" onClick={() => setStep(0)} />
                    </Box>
                    <Button type="submit" primary label="Continue" />
                </Box>
            </Form>
        </Box>
    );
};

export default AuthorityForm;