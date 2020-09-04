import React, { useContext, useState } from "react";
import { Link } from "react-router-dom";

import { Box, Text, Button, Form } from "grommet";
import locale from "../locale";

import { ConsoleLogo } from "../../index";
import { LoginInput, PasswordInput } from "../components";
import { AuthenticationContext } from "../AuthenticationContext";

const Login = () => {
    const [ email, setEmail ] = useState('');
    const [ password, setPassword ] = useState('');
    const { message, errors, loading, login } = useContext(AuthenticationContext)

    return (
        <Form onSubmit={() => login(email, password)}>
            <Box gap="medium" align="center" justify="center">
                <Box gap="small" width="medium">
                    <Box margin={{vertical:'medium'}}>
                        <ConsoleLogo />
                    </Box>

                    <LoginInput value={email} disabled={loading}
                                onChange={(value) => setEmail(value)}
                    />
                    <PasswordInput value={password} disabled={loading}
                                   onChange={(value) => setPassword(value)}
                    />
                    <Box direction="row">
                        <Button type="submit" primary label={locale.login} disabled={loading} />
                        <Box pad="xsmall" margin={{horizontal: "small"}}>
                            <Link to="/password/reset">{locale.forgot}</Link>
                        </Box>
                    </Box>
                    <Box direction="row" gap="small" border={{side:'top',color:'brand',size:'small'}} pad={{top:'medium'}}>
                        <Text>
                            You don't have an account?
                        </Text>
                        <Link to="/signup">Signup</Link>
                    </Box>
                    {message && <Text color="status-critical">{message}</Text>}
                    {errors && Object.entries(errors).map(v => v[1].join(' '))}
                </Box>
            </Box>
        </Form>
    );

}

export default Login;


