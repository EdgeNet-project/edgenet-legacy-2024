import React from "react";
import {Box, FormField, Text} from "grommet";


const SignupAuthority = () =>
    <Box>
        <Text color="dark-2" margin={{bottom:'small'}}>
            Please complete with the information of the institution you are part of
        </Text>
        <FormField label="Institution full name" name="fullname" required />
        <FormField label="Institution shortname or initials" name="shortname" required validate={{ regexp: /^[a-z]/i }} />
        <FormField label="Address" name="street" required />
        <Box direction="row" gap="small">
            <FormField label="ZIP code" name="zip" required />
            <FormField label="City" name="city" required />
        </Box>
        <Box direction="row" gap="small">
            <FormField label="Region" name="region" />
            <FormField label="Country" name="country" required />
        </Box>
        <FormField label="Web page" name="url" required />
    </Box>

export default SignupAuthority;