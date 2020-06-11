import React from "react";
import axios from "axios";

import {Box, Heading, Text, Image} from "grommet";

import { NavigationAnchor } from "../components/navigation";
import { AuthorityForm, AuthoritySelect, UserForm } from "./registration";

const RegistrationSubmitting = () =>
    <Box pad={{vertical:"large"}}>
        <Text>
            Submitting
        </Text>
    </Box>;

const RegistrationError = () =>
    <Box pad={{vertical:"large"}}>
        <Text>
            An error has occurred, please contact support.
        </Text>
    </Box>;

const RegistrationSuccess = () =>
    <Box pad={{vertical:"large"}}>
        <Text>
            Your request has been registered, thank you.
            You will receive an email ...
        </Text>
    </Box>;

class RegistrationPanel extends React.Component {

    constructor(props) {
        super(props);

        this.state = {
            authority: null,
            user: null,

            step: 0

        };

        this.setStep = this.setStep.bind(this);
        this.setAuthority = this.setAuthority.bind(this);
        this.setUser = this.setUser.bind(this);

        this.submit = this.submit.bind(this);
        this.error = this.error.bind(this);

    }

    createName(str) {
        return str.replace(/[^a-z0-9]+/gi, '').replace(/^-*|-*$/g, '').toLowerCase();
    }


    error(message) {
        this.setState({ message: message })
    }

    setAuthority(value) {
        console.log(value)
        this.setState({
            authority: value
        }, () => this.setStep(2))
    }

    setUser(value) {

        this.setState({
            user: value,
            step: 3
        }, this.submit)
    }


    submit() {
        const { user, authority } = this.state;
        axios.post(
                'register', {
                    ...user, authority: authority
            })
                .then(({data}) => this.setState({step: 4}))
                .catch((error) => {
                    this.setState({
                        step: 2, message: error.response.data.message
                    });

                    if (error.response) {
                        console.log('a',error.response.data);
                    } else if (error.request) {
                        console.log('b',error.request);
                    } else {
                        console.log('client error');
                    }
                })




    }

    setStep(step) {
        this.setState({step: step})
    }

    steps() {
        const { step, authority, user } = this.state;
        switch(step) {
            default:
            case 0:
                return <AuthoritySelect authority={authority}
                                        setAuthority={this.setAuthority}
                                        setStep={this.setStep} />;
            case 1:
                return <AuthorityForm authority={authority}
                                      setAuthority={this.setAuthority}
                                      setStep={this.setStep} />;
            case 2:
                return <UserForm user={user} setUser={this.setUser} setStep={this.setStep} />;
            case 3:
                return <RegistrationSubmitting />;
            case 4:
                return <RegistrationSuccess />;

        }
    }

    render() {
        const { authority } = this.state;

        const message = '';


        return (
            <Box align="center" justify="center">
                <Image style={{maxWidth:'25%',margin:'50px auto'}} src="images/edgenet.png" alt="EdgeNet" />
                <Box width="medium" pad={{bottom:"small"}} align="center" border={{side:"bottom",color:"brand",size:"small"}}>
                    <Heading level="3">
                        Create your EdgeNet Account
                    </Heading>
                </Box>
                {authority &&
                <Box width="medium" pad={{vertical:"small"}} align="start">
                    <Heading level="4">Your institution</Heading>
                    {authority.name} - {authority.shortname} <br />
                    {authority.address} <br />
                    {authority.zipcode} {authority.city} <br />
                    {authority.country}
                </Box>
                }
                <Box width="medium">
                    {this.steps()}

                    {message && <Text color="status-critical">{message}</Text>}
                </Box>
                <Box width="medium" pad={{top:"medium"}} align="center" border={{side:"top",color:"brand",size:"small"}}>
                    <NavigationAnchor label="Go back to login" path="/" />
                </Box>
            </Box>
        )
    }
}

export default RegistrationPanel;