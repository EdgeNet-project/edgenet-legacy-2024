import React from 'react';
import axios from "axios";

import {Box, Heading, Text, Button, Anchor} from "grommet";

import { NavigationAnchor } from "../components/navigation";
import EdgenetLogo from "../components/utils/EdgenetLogo";
import SelectInstitution from "./registration/SelectInstitution";
import SubmitInstitution from "./registration/SubmitInstitution";
import SubmitPerson from "./registration/SubmitPerson";
import Institution from "./registration/Institution";

class RegistrationPanel extends React.Component {

    constructor(props) {
        super(props);

        this.state = {
            step: 1,
            submitInstitution: false,
            validatedInstitution: false,
            institution: {},

            validatedPerson: false,
            person: {}
        };

        this.steps = this.steps.bind(this);
        this.submit = this.submit.bind(this);
        this.error = this.error.bind(this);

        this.setSubmitInstitution = this.setSubmitInstitution.bind(this);
        this.setSelectInstitution = this.setSelectInstitution.bind(this);
        this.onChangeInstitution = this.onChangeInstitution.bind(this);
        this.onSelectInstitution = this.onSelectInstitution.bind(this);
        this.validateInstitution = this.validateInstitution.bind(this);
        this.onChangePerson = this.onChangePerson.bind(this);
        this.validatePerson = this.validatePerson.bind(this);
    }

    createName(str) {
        return str.replace(/[^a-z0-9]+/gi, '').replace(/^-*|-*$/g, '').toLowerCase();
    }

    prepareRequest() {
        const { institution, person } = this.state;

        return {
            apiVersion: 'apps.edgenet.io/v1alpha',
            kind: 'SiteRegistrationRequest',
            metadata: {
                name: this.createName(institution.shortname)
            },
            spec: {
                fullname: institution.fullname,
                shortname: institution.shortname,
                url: institution.url,
                address: institution.address,
                contact: {
                    username: this.createName(person.firstname+person.lastname),
                    firstname: person.firstname,
                    lastname: person.lastname,
                    email: person.email,
                    phone: person.phone
                }
            }
        }

    }

    error(message) {
        this.setState({ message: message })
    }
    submit() {

        this.setState({
                step: 3
            }, () => axios.post(
            'siteregistrationrequests', this.prepareRequest(),
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
                })
        )



    }

    setSubmitInstitution() {
        this.setState({
            submitInstitution: true,
            institution: {},
            validatedInstitution: false
        })
    }

    setSelectInstitution() {
        this.setState({
            submitInstitution: false,
            institution: {},
            validatedInstitution: false
        })
    }

    onChangeInstitution(value) {
        // bugfix for Grommet Form event leaking
        if (value.target !== undefined) return null;

        this.setState({
            institution: value,
            validatedInstitution: this.validateInstitution(value)
        })
    }

    onChangePerson(value) {
        // bugfix for Grommet Form event leaking
        if (value.target !== undefined) return null;

        this.setState({
            person: value,
            validatedPerson: this.validatePerson(value)
        })
    }

    onSelectInstitution(value) {
        this.setState({
            institution: value,
            validatedInstitution: true
        })
    }

    validateInstitution(value) {

        if (!value.fullname) {
            return false
        }

        if (!value.shortname) {
            return false
        }

        if (!value.address) {
            return false
        }

        if (!value.url) {
            return false
        }

        return true;
    }

    validatePerson(value) {
        return true;
    }

    steps() {
        const { step, submitInstitution, validatedInstitution, institution, validatedPerson } = this.state;

        switch(step) {
            case 1:
                return (
                    <Box>
                        {submitInstitution ?
                            <SubmitInstitution value={institution} onChange={this.onChangeInstitution}/> :
                            <SelectInstitution selected={institution.name} onSelect={this.onSelectInstitution} />}
                        <Box pad={{vertical:"medium"}} direction="row" justify="end" align="center">
                            <Box pad={{right:"small"}} margin={{right:"small"}}>
                                {submitInstitution ? <Anchor alignSelf="start" label="Cancel" onClick={this.setSelectInstitution} /> :
                                    <Anchor alignSelf="start" label="My institution is not on the list" onClick={this.setSubmitInstitution} />}
                            </Box>
                            <Button disabled={!validatedInstitution} primary label="Continue" onClick={() => this.setState({step: 2})} />
                        </Box>
                    </Box>
                );
            case 2:
                return (
                    <Box>
                        <Box border={{side:"bottom",color:"light-2",size:"xsmall"}}>
                            {institution && <Institution item={institution} />}
                        </Box>
                        <SubmitPerson onChange={this.onChangePerson} />
                        <Box pad={{vertical:"medium"}} direction="row" justify="end" align="center">
                            <Box pad={{right:"medium"}} margin={{right:"medium"}}>
                                <Anchor alignSelf="start" label="Back" onClick={() => this.setState({step: 1})} />
                            </Box>
                            <Button disabled={!validatedPerson} primary label="Continue" onClick={this.submit} />
                        </Box>
                    </Box>
                );
            case 3:
                return (
                    <Box pad={{vertical:"large"}}>
                        <Text>
                            Submitting
                        </Text>
                    </Box>
                );
            case 4:
                return (
                    <Box pad={{vertical:"large"}}>
                        <Text>
                            Your request has been registered, thank you.
                            You will receive an email ...
                        </Text>
                    </Box>
                );
            case 5:
                return (
                    <Box pad={{vertical:"large"}}>
                        <Text>
                            An error has occurred, please contact support.
                        </Text>
                    </Box>
                );
        }
    }

    render() {
        const { step } = this.state;
        const message = '';
        const steps = [
            'Institution',
            'Account'
        ];

        return (
            <Box align="center" justify="center">
                <EdgenetLogo />
                <Heading level="3">
                    Create your EdgeNet Account
                </Heading>
                <Box width="medium">
                    <Box direction="row" border={{side:"bottom", color:"brand"}} justify="evenly" width="100%" gap="small" pad="small">
                        {steps.map((s, k) =>
                            <Box key={s + k} direction="row" gap="xsmall" pad={{vertical:"xsmall"}}>
                                <Box background={step === (k+1) ? 'brand' : 'light-2'} round pad={{horizontal:"xsmall"}}>{k + 1}</Box> {s}</Box>)}
                    </Box>
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