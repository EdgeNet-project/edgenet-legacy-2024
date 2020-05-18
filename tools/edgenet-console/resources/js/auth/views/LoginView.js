import React from "react";
import { Link } from "react-router-dom";

import { Box, Text, Button, Image, Form } from "grommet";
import { LoginInput, PasswordInput } from "../components";

import { AuthContext } from "../AuthContext";

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
        const { title, logo } = this.props;

        return (
            <Form onSubmit={() => login(email, password)}>
                <Box gap="medium" align="center" justify="center">
                    <Box gap="small" margin={{top:"large"}}>
                        {logo && <Image style={{maxWidth:'25%',margin:'50px auto'}} src={logo} alt={title} />}
                        {title ? title : "Admin"}
                    </Box>

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
                    {message && <Text color="status-critical">{message}</Text>}
                </Box>
            </Form>
        );

    }
}

LoginView.contextType = AuthContext;

export default LoginView;
