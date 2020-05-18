import React from 'react';

import { Box, Text, Button, Image, Form } from 'grommet';
import { LoginInput } from '../components';

import { AuthContext } from "../AuthContext";

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
        const { title, logo } = this.props;

        return (
            <Form onSubmit={() => sendResetLink(email)}>
                <Box gap="medium" align="center" justify="center">
                    <Box gap="small" margin={{top:"large"}}>
                        {logo && <Image style={{maxWidth:'25%',margin:'50px auto'}} src={logo} alt={title} />}
                        {title ? title : "Forgot Password"}
                    </Box>

                    <Box gap="small" width="medium">
                        <LoginInput value={email} disabled={loading}
                                    onChange={(value) => this.setState({email: value})}
                        />
                    </Box>
                    <Box direction="row">
                        <Button type="submit" primary label="Submit" disabled={loading} />
                    </Box>
                    {message && <Text color="status-critical">{message}</Text>}
                </Box>
            </Form>
        );

    }
}

ForgotPasswordView.contextType = AuthContext;

export default ForgotPasswordView;
