import React from 'react';
import { Box, Text, Button, Form } from 'grommet';
import { LoginInput, PasswordInput } from "../../components/form";

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
        const { message, loading, sendResetLink } = this.context;

        return (
            <Form onSubmit={() => sendResetLink(email)}>
                <Box gap="medium" align="center" justify="center" width="medium">
                    <Text>
                        Specify your E-Mail address and your new password.
                    </Text>
                    <Box width="medium" gap="small">
                        <LoginInput value={email} disabled={loading}
                                    onChange={(value) => this.setState({email: value})}
                        />
                        <PasswordInput value={password} disabled={loading}
                                       onChange={(value) => this.setState({password: value})}
                        />
                        <PasswordInput value={password} disabled={loading}
                                       onChange={(value) => this.setState({password: value})}
                        />
                    </Box>
                    <Box direction="row">
                        <Button type="submit" primary label="Reset" disabled={loading} />
                    </Box>
                    <NavigationAnchor label="Go back to the login page" path="/" />
                    {message && <Text color="status-critical">{message}</Text>}
                </Box>
            </Form>
        );

    }
}

PasswordResetForm.contextType = UserContext;

export default PasswordResetForm;
