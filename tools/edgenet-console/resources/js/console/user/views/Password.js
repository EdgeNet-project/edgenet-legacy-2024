import React from "react";
import axios from "axios";

import { Box, Heading, Text, Form, FormField, TextArea, Button } from "grommet";
import { ClearOption } from "grommet-icons";
import { PasswordInput } from "../../authentication/components";

class Password extends React.Component {

    constructor(props) {
        super(props);

        this.state = {
            value: {
                old: '', password_1: '', password_2: ''
            },
            disabled: true,
            message: '',
            loading: false,
        };

        this.submit = this.submit.bind(this);
        this.save = this.save.bind(this);
        this.enable = this.enable.bind(this);
    }

    submit({value}) {
        this.setState({
            loading: true
        }, () => this.save(value));
    }
    save(value) {
        const { password_1, password_2, old } = value;

        if (!old) {
            this.setState({
                message: 'Provide your old password',
                loading: false,
            });
            return;
        }

        if (!password_1 || (password_1 !== password_2)) {
            this.setState({
                message: 'The new password doesn\'t match',
                loading: false,
            });
            return;
        }

        axios.post('user/password', {
            password: password_1,
            old: old
        })
            .then(response => {
                let { data, meta } = response.data;
                this.setState({
                    value: {
                        old: '', password_1: '', password_2: ''
                    },
                    message: 'Your password has been successfully updated',
                    loading: false,
                });
            })
            .catch(error => {
                this.setState({
                    message: 'Error updating your password',
                    loading: false,
                });
            });
    }

    enable(value) {
        const { loading } = this.state;
        this.setState({
            disabled: (!value.password_1 || !value.password_2 || !value.old || loading)
        });
    }

    clear() {}

    render() {
        let { value, disabled, message } = this.state;

        return (
            <Form value={value} onSubmit={this.submit} onChange={this.enable}>
                <Box pad="medium" gap="small" align="start">
                    <Heading level="3" margin="none">Update my password</Heading>
                    <Text>

                    </Text>
                    <FormField plain label="Current password" name="old" component={PasswordInput} />
                    <FormField plain label="New password" name="password_1" component={PasswordInput} />
                    <FormField plain label="Repeat" name="password_2" component={PasswordInput} />

                    <Box direction="row" gap="small">
                        <Button type="submit" primary label="Submit" disabled={disabled} />
                        <Text margin="xsmall">
                            {message}
                        </Text>
                    </Box>
                </Box>
            </Form>
        );
    }
}


export default Password;
