import React from 'react';

import { Box, Text, Button, Image, Form } from 'grommet';
import { LoginInput } from '../components';

import { AuthContext } from "../AuthContext";
import Header from "./Header";
import Footer from "./Footer";

class ForgotPasswordView extends React.Component {

    constructor(props) {
        super(props);
        this.state = {
            email: '',
            password: ''
        };
    }

    render() {
        const { email } = this.state;
        const { message, loading, sendResetLink } = this.context;

        return (
            <Form onSubmit={() => sendResetLink(email)}>
                <Box gap="medium" align="center" justify="center">
                    <Header title="Password reset" />

                    <Box gap="small" width="medium">
                        <LoginInput value={email} disabled={loading}
                                    onChange={(value) => this.setState({email: value})}
                        />
                    </Box>
                    <Box direction="row">
                        <Button type="submit" primary label="Submit" disabled={loading} />
                    </Box>
                    <Footer />
                    {message && <Text color="status-critical">{message}</Text>}
                </Box>
            </Form>
        );

    }
}

ForgotPasswordView.contextType = AuthContext;

export default ForgotPasswordView;
