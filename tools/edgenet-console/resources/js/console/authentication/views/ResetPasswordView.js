import React, {useContext, useState} from "react";
import { Link, useParams } from "react-router-dom";
import { Box, Heading, Text, Button, Form } from "grommet";
import { LoginInput, PasswordInput } from "../components";

import locale from "../locale";
import {ConsoleLogo} from "../../index";

import { AuthenticationContext } from "../AuthenticationContext";

const ResetPasswordView = () => {
    const [ email, setEmail ] = useState((new URLSearchParams(window.location.search)).get('email') || '');
    const [ password, setPassword ] = useState('');
    const [ password_confirmation, setPasswordConfirmation ] = useState('');
    const { message, errors, loading, resetPassword, prefix } = useContext(AuthenticationContext);
    const { token } = useParams();

    if (!token) {
        return 'no token';
    }

    return (
        <Form onSubmit={() => resetPassword(email, token, password, password_confirmation)}>
            <Box gap="medium" align="center" justify="center">
                <Box gap="small" width="medium">
                    <Box margin={{vertical:'medium'}}>
                        <ConsoleLogo />
                    </Box>
                    <Heading level="2" size="small" margin="none">{locale.resetTitle}</Heading>
                    <Text size="small">
                        {locale.resetText}
                    </Text>
                    <LoginInput value={email} disabled={loading} required onChange={setEmail} />
                    <PasswordInput disabled={loading} placeholder={locale.resetNewPassword} required
                                   onChange={setPassword} />
                    <PasswordInput disabled={loading} placeholder={locale.resetConfirmPassword} required
                                   onChange={setPasswordConfirmation} />

                    <Button alignSelf="start" type="submit" primary label={locale.recoverSubmit} disabled={loading} />
                    <Link to='/'>{locale.linkHome}</Link>
                    {message && <Text color="status-critical">{message}</Text>}
                    {errors && Object.entries(errors).map(v => v[1].join(' '))}
                </Box>
            </Box>
        </Form>
    );

}

export default ResetPasswordView;