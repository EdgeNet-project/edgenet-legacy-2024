import React from "react";
import {Box, TextInput, FormField} from "grommet";


const SignupUser = () =>
    <Box>
        <FormField label="Firstname" htmlfor="firstname" name="firstname" required validate={{ regexp: /^[a-z]/i }}>
            <TextInput id="firstname" name="firstname" />
        </FormField>
        <FormField label="Lastname" name="lastname" htmlfor="lastname" required validate={{ regexp: /^[a-z]/i }}>
            <TextInput id="lastname" name="lastname" />
        </FormField>
        <FormField label="Phone" name="phone" htmlfor="phone">
            <TextInput id="phone" name="phone" />
        </FormField>
        <FormField label="E-Mail" name="email" htmlfor="email" required>
            <TextInput id="email" name="email" />
        </FormField>
        <FormField label="Password" name="email" htmlfor="password" required>
            <TextInput id="password" name="password" type="password" />
        </FormField>
        <FormField label="Password confirmation" name="password_confirmation" htmlfor="password_confirmation" required>
            <TextInput id="password_confirmation" name="password_confirmation" type="password" />
        </FormField>
    </Box>

export default SignupUser;