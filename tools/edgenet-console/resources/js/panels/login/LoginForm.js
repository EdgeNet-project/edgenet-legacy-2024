import React from 'react';

import { Box, Text, Button, Image, Form } from 'grommet';
import { LoginInput, PasswordInput } from '../../components/form';
import { NavigationAnchor } from "../../components/navigation";

import { UserContext } from "../../components/user";

class LoginForm extends React.Component {

    constructor(props) {
        super(props);
        this.state = {
            email: '',
            password: ''
        };
    }

    render() {
        const { email, password } = this.state;
        const { message, loading, login, passwordResetLink } = this.context;

        return (
                <Form onSubmit={() => login(email, password)}>
                    <Box gap="medium" align="center" justify="center">
                        <Image style={{maxWidth:'25%',margin:'50px auto'}} src="images/edgenet.png" alt="EdgeNet" />
                        <Box gap="small" width="medium">
                            <LoginInput value={email} disabled={loading}
                                        onChange={(value) => this.setState({email: value})}
                            />
                            <PasswordInput value={password} disabled={loading}
                                           onChange={(value) => this.setState({password: value})}
                            />
                        </Box>
                        <Box direction="row">
                            <Box pad="xsmall" margin={{horizontal: "small"}}>
                                {passwordResetLink && <NavigationAnchor label="forgot password?" path={passwordResetLink} />}
                            </Box>
                            <Button type="submit" primary label="Log in" disabled={loading} />
                        </Box>
                        {message && <Text color="status-critical">{message}</Text>}
                        <Box width="medium" pad="medium" gap="medium" border={{side:'top', color:'brand', size:'small'}} align="center">
                            <Box>
                                <NavigationAnchor label="Create an account" path="/signup" />
                            </Box>
                            {/*<Box>*/}
                            {/*    <NavigationAnchor label="File a site registration" path={passwordResetLink} />*/}
                            {/*</Box>*/}

                        </Box>
                    </Box>
                </Form>
        );

    }
}

LoginForm.contextType = UserContext;

export default LoginForm;
