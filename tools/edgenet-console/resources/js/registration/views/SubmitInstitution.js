import React from "react";
import {Box, Text, Form, FormField} from "grommet";

const SubmitInstitution = ({value, onChange}) =>
    <Box width="medium">
        <Box pad={{vertical:"medium"}}>
            <Text color="dark-2">Please complete with the information of the institution you are part of</Text>
        </Box>
        <Box >
            <Form value={value} onChange={onChange}>
            <FormField label="Institution full name" name="fullname" required validate={{ regexp: /^[a-z]/i }} />
            <FormField label="Institution shortname or initials" name="shortname" required validate={{ regexp: /^[a-z]/i }} />
            <FormField label="Address" name="address" required validate={{ regexp: /^[a-z]/i }} />
            <FormField label="Web page" name="url" required validate={{ regexp: /^[a-z]/i }} />
            </Form>
        </Box>
    </Box>;

export default SubmitInstitution;