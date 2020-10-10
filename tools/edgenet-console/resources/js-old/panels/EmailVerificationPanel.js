import React from 'react';
import axios from "axios";

import {Box, Heading, Text, Button, Anchor, Form, FormField} from "grommet";

import EdgenetLogo from "../components/utils/EdgenetLogo";
import {NavigationAnchor} from "../components/navigation";


class EmailVerificationPanel extends React.Component {

    constructor(props) {
        super(props);
        this.state = {
            token: '',
            loading: false,
            verified: false
        };
        this.verify = this.verify.bind(this);
    }

    componentDidMount() {
        const { match } = this.props;
        if (match.params.token) {
            this.setState({token: match.params.token})
        }
    }

    verify({ value }) {
        console.log(value.token)

        this.setState({
            loading: true
        }, () => axios.patch(
            'namespaces/site-edgenet/emailverifications/' + value.token,
            [{"op": "replace", "path": "/spec/verified", "value": true}],
            { headers: {'content-type': 'application/json-patch+json'}}
        )
            .then(({data}) => this.setState({step: 4}))
            .catch((error) => {
                this.setState({
                    step: 5,
                });

                if (error.response) {
                    console.log(error.response.data);
                } else if (error.request) {
                    console.log(error.request);
                } else {
                    console.log('client error');
                }
            }))
    }

    render() {
        const { token } = this.state;

        return (
            <Box align="center" justify="center">
                <EdgenetLogo />
                <Heading level="3">
                    Verify your email
                </Heading>
                <Form value={{token: token}} onSubmit={this.verify}>
                    <FormField name="token" label="Email verification token" />
                    <Box pad={{vertical:"medium"}} direction="row" justify="end" align="center">
                        <Button type="submit" primary label="Continue" onClick={this.verify} />
                    </Box>
                </Form>
                <Box width="medium" pad={{top:"medium"}} align="center" border={{side:"top",color:"brand",size:"small"}}>
                    <NavigationAnchor label="Go back to login" path="/" />
                </Box>
            </Box>
        );
    }
}

export default EmailVerificationPanel;