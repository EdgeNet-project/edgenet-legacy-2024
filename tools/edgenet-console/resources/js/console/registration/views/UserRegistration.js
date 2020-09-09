import React, {useState, useContext} from "react";
import {Anchor, Box, Button, Form, FormField, Text, TextInput} from "grommet";
import {Link} from "react-router-dom";
import { ConsoleLogo } from "../../index";

import { RegistrationContext } from "../RegistrationContext";
import AuthorityRegistration from "./AuthorityRegistration";
import AuthoritySelect from "./AuthoritySelect";
import Succes from "./Succes";

const UserRegistration = () => {
    const [ createAuthority, setCreateAuthority ] = useState(false);
    const { submitRegistration, loading, message, errors, success } = useContext(RegistrationContext);

    if (success) {
        return <Succes />;
    }

    return (
        <Box align="center">
            <Box gap="medium" alignSelf="center" width="medium" alignContent="center" align="stretch">
                <Box margin={{vertical:'medium'}}>
                    <ConsoleLogo />
                </Box>
                <Box border={{side: 'bottom', color: 'brand', size: 'small'}}
                     pad={{vertical: 'medium'}} gap="small">

                    {createAuthority ? <AuthorityRegistration /> : <AuthoritySelect />}

                    <Anchor onClick={() => setCreateAuthority(!createAuthority)}>
                        {createAuthority ? "I want to select an existing institution" : "My institution is not on the list" }
                    </Anchor>
                </Box>


                <Box border={{side:'bottom',color:'brand',size:'small'}}>
                    <Form onSubmit={submitRegistration}>
                        <FormField label="Firstname" htmlfor="firstname" error={errors.firstname} name="firstname" required validate={{ regexp: /^[a-z]/i }}>
                            <TextInput id="firstname" name="firstname" />
                        </FormField>
                        <FormField label="Lastname" error={errors.lastname} name="lastname" htmlfor="lastname" required validate={{ regexp: /^[a-z]/i }}>
                            <TextInput id="lastname" name="lastname" />
                        </FormField>
                        <FormField label="Phone" error={errors.phone} name="phone" htmlfor="phone">
                            <TextInput id="phone" name="phone" />
                        </FormField>
                        <FormField label="E-Mail" error={errors.email} name="email" htmlfor="email" required>
                            <TextInput id="email" name="email" />
                        </FormField>
                        <FormField label="Password" error={errors.password} name="password" htmlfor="password" required>
                            <TextInput id="password" name="password" type="password" />
                        </FormField>
                        <FormField label="Password confirmation" error={errors.password_confirmation} name="password_confirmation" htmlfor="password_confirmation" required>
                            <TextInput id="password_confirmation" name="password_confirmation" type="password" />
                        </FormField>

                        <Box direction="row" pad={{vertical:'medium'}} justify="between" align="center">
                            <Link to="/migrate">Migrate my PlanetLab Europe account</Link>
                            <Button disabled={loading} type="submit" primary label="Register" />
                        </Box>
                    </Form>
                </Box>
                {message && <Text color="status-critical">{message}</Text>}
                <Box direction="row" >
                    <Link to="/">Go back to the login page</Link>
                </Box>
            </Box>
        </Box>
    );
}

export default UserRegistration;