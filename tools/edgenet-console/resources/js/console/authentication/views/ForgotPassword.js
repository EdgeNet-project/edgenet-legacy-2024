import React, { useContext, useState } from 'react';
import {Link} from "react-router-dom";

import { Box, Heading, Text, Button, Form } from 'grommet';
import { LoginInput } from '../components';

import locale from "../locale";

import { AuthenticationContext } from "../AuthenticationContext";
import { ConsoleLogo } from "../../index";

const ForgotPasswordView = () => {
    const [email, setEmail] = useState('');
    const {message, errors, loading, sendResetLink, prefix} = useContext(AuthenticationContext)

    return (
        <Form onSubmit={() => sendResetLink(email)}>
            <Box gap="medium" align="center" justify="center">
                <Box gap="small" width="medium">
                    <Box margin={{vertical:'medium'}}>
                        <ConsoleLogo />
                    </Box>
                    <Heading level="2" size="small" margin="none">{locale.recoverTitle}</Heading>
                    <Text size="small">
                        {locale.recoverText}
                    </Text>
                    <LoginInput value={email} disabled={loading} onChange={setEmail}/>
                    <Button alignSelf="start" type="submit" primary label={locale.recoverSubmit} disabled={loading}/>
                    <Link to="/">{locale.linkHome}</Link>
                    {message && <Text color="status-critical">{message}</Text>}
                    {errors && Object.entries(errors).map(v => v[1].join(' '))}
                </Box>
            </Box>
        </Form>
    );
}

export default ForgotPasswordView;
