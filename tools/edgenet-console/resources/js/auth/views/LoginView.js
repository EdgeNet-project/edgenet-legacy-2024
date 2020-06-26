import React from "react";
import { Link } from "react-router-dom";

import { Box, Text, Button, Image, Form } from "grommet";
import { LoginInput, PasswordInput } from "../components";

import { AuthContext } from "../AuthContext";
import Header from "./Header";

class LoginView extends React.Component {

    constructor(props) {
        super(props);
        this.state = {
            email: '',
            password: ''
        };
    }

    render() {
        const { email, password } = this.state;
        const { message, loading, login } = this.context;

        return (
            <Form onSubmit={() => login(email, password)}>
                <Box gap="medium" align="center" justify="center">
                    <Header title="Login" />

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
                            <Link to="/password/reset">forgot password?</Link>
                        </Box>
                        <Button type="submit" primary label="Log in" disabled={loading} />
                    </Box>
                    <Box direction="row" gap="small" border={{side:'top',color:'brand',size:'small'}} pad={{top:'medium'}}>
                        <Text>
                        You don't have an account?
                        </Text>
                        <Link to="/signup">Signup</Link>
                    </Box>
                    {message && <Text color="status-critical">{message}</Text>}
                </Box>
            </Form>
        );

    }
}

LoginView.contextType = AuthContext;

export default LoginView;
