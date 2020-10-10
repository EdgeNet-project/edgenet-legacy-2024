import React from 'react';
import { Box, Text, Button, Form } from 'grommet';
import { LoginInput } from "../../components/form";

import { UserContext } from "../../components/user/UserContext";
import { NavigationAnchor } from "../../components/navigation";

class PasswordResetForm extends React.Component {

    constructor(props) {
        super(props);
        this.state = {
            email: ''
        };
    }

    render() {
        const { email } = this.state;
        const { message, loading, sendResetLink, loginPath } = this.context;

        return (
            <Form onSubmit={() => sendResetLink(email)}>
                <Box gap="medium" align="center" justify="center" width="medium">
                    <Text>
                        Specify your E-Mail address, you will receive an email with the instructions on how to reset your password.
                    </Text>
                    <Box width="medium">
                        <LoginInput value={email} disabled={loading}
                                    onChange={(value) => this.setState({email: value})}
                        />
                    </Box>
                    <Box direction="row">
                        <Button type="submit" primary label="Send reset link" disabled={loading} />
                    </Box>
                    <NavigationAnchor label="Go back to the login page" path={loginPath} />
                    {message && <Text color="status-critical">{message}</Text>}
                </Box>
            </Form>
        );

    }
}

PasswordResetForm.contextType = UserContext;

export default PasswordResetForm;
