import React from "react";
import {Anchor, Box, Button, Form, FormField, TextInput, Text} from "grommet";

const UserForm = ({user, setUser, setStep}) => {
    const [value, setValue] = React.useState(user ? user : {
        firstname: "",
        lastname: "",
        email: "",
        phone: "",
        bio: "",
        password: "",
        password_confirmation: ""
    });

    return (
        <Box width="medium">
            <Form value={value}
                  onChange={nextValue => setValue(nextValue)}
                  onReset={() => setValue({})}
                  onSubmit={({ value }) => setUser(value)}>
                <Box pad={{vertical: "medium"}}>
                    <Text color="dark-2">
                        Please complete with your information
                    </Text>
                </Box>
                <Box>
                    <Box direction="row" gap="small">
                        <FormField label="Firstname" name="firstname" required validate={{regexp: /^[a-z]/i}} />
                        <FormField label="Lastname" name="lastname" required validate={{regexp: /^[a-z]/i}} />
                    </Box>
                    <FormField label="E-Mail" name="email" required />
                    <FormField label="Phone" name="phone" required />
                    <FormField label="Password" name="password">
                        <TextInput required name="password" type="password" />
                    </FormField>
                    <FormField label="Confirm" name="password_confirmation">
                        <TextInput required name="password_confirmation" type="password" />
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
}

export default UserForm;