import React from 'react';

import { Box, Text, Button, Image, Form } from 'grommet';
import { LoginInput, PasswordInput } from '../components';

import { AuthContext } from "../AuthContext";

class ResetPasswordView extends React.Component {

    constructor(props) {
        super(props);
        this.state = {
            email: '',
            password: '',
            password_confirmation: '',
        };
    }

    componentDidMount() {
        this.setState({
            email: (new URLSearchParams(window.location.search)).get('email') || ''
        })
    }

    render() {
        const { email, password, password_confirmation } = this.state;
        const { message, loading, resetPassword } = this.context;
        const { title, logo, token } = this.props;

        if (!token) {
            return 'no token';
        }

        return (
            <Form onSubmit={() => resetPassword(email, token, password, password_confirmation)}>
                <Box gap="medium" align="center" justify="center">
                    <Box gap="small" margin={{top:"large"}}>
                        {logo && <Image style={{maxWidth:'25%',margin:'50px auto'}} src={logo} alt={title} />}
                        {title ? title : "Reset Password"}
                    </Box>

                    <Box gap="small" width="medium">
                        <LoginInput value={email} disabled={loading}
                                    onChange={(value) => this.setState({email: value})}
                        />
                        <PasswordInput disabled={loading} placeholder="New password"
                                       onChange={(value) => this.setState({password: value})} />
                        <PasswordInput disabled={loading} placeholder="Confirm password"
                                       onChange={(value) => this.setState({password_confirmation: value})} />
                    </Box>
                    <Box direction="row">
                        <Button type="submit" primary label="Reset Password" disabled={loading} />
                    </Box>
                    {message && <Text color="status-critical">{message}</Text>}
                </Box>
            </Form>
        );

    }
}

ResetPasswordView.contextType = AuthContext;

export default ResetPasswordView;
