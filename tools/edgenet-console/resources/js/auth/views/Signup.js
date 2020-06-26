import React from "react";
import {Box, Form, Text, Button, Anchor} from "grommet";
import {Link} from "react-router-dom";
import Select from 'react-select';
import axios from "axios";

import { EdgenetContext } from "../../edgenet";
import SignupSucces from "./SignupSucces";
import SignupUser from "./SignupUser";
import SignupAuthority from "./SignuAuthority";
import Loading from "./Loading";
import Header from "./Header";

class Signup extends React.Component {

    constructor(props) {
        super(props);
        this.state = {
            authorities: [],

            value: {
                firstname: '',
                lastname: '',
                phone: '',
                email: '',
                password: '',
                password_confirmation: '',

                fullname: '',
                shortname: '',
                street: '',
                zip: '',
                city: '',
                region: '',
                country: '',
                url: ''
            },
            authority: null,
            signupAuthority: false,

            message: null,
            error: null,
            loading: false
        }

        this.handleChange = this.handleChange.bind(this);
        this.signupAuthority = this.signupAuthority.bind(this);
        this.getAuthorities = this.getAuthorities.bind(this);
        this.selectAuthority = this.selectAuthority.bind(this);
        this.signup = this.signup.bind(this);
    }

    componentDidMount() {
        this.getAuthorities()
    }

    handleChange(value) {
        this.setState({value: value})
    }

    signup({value}) {
        const { authority } = this.state;
        console.log(value)
        this.setState({
            loading: true
        }, () =>
                axios.post('/signup', {
                    ...value,
                    authority: authority
                })
                    .then(({data}) => this.setState({
                        loading: false,
                        success: true
                    }, () => console.log(data)))
                    .catch(error => {
                        this.setState({
                            loading: false,
                            message: error.message
                        });
                        if (error.response) {
                            console.log(error.response.data);
                        } else if (error.request) {
                            console.log(error.request);
                        } else {
                            console.log('client error');
                            console.log(error);
                        }
                    })
        );
    }

    signupAuthority() {
        const { signupAuthority } = this.state;

        this.setState({
            signupAuthority: !signupAuthority,
            authority: null
        })
    }

    getAuthorities() {
        const { api } = this.context;
        axios.get(api + '/apis/apps.edgenet.io/v1alpha/authorities')
            .then(({data}) =>
                this.setState({
                    authorities: data.items.map(item => {
                        return {
                            value: item.metadata.name, label: item.spec.fullname + ' ('+item.spec.shortname+')'
                        }
                    }),
                }, () => console.log(data))
            );
    }

    selectAuthority({value}) {
        this.setState({
            authority: value,
            signupAuthority: null
        })
    }

    render() {
        const { value, authorities, authority, signupAuthority, message, loading, success } = this.state;

        if (success) {
            return <SignupSucces />;
        }

        if (loading) {
            return <Loading />;
        }

        return (
            <Box align="center">
                <Header />
                <Form onSubmit={this.signup} onChange={this.handleChange} value={value}>
                    <Box gap="medium" alignSelf="center" width="medium" alignContent="center" align="stretch">
                            <Box border={{side: 'bottom', color: 'brand', size: 'small'}}
                                 pad={{vertical: 'medium'}} gap="small">

                                {signupAuthority ? <SignupAuthority />
                                    :
                                    <Select placeholder="Select your institution"
                                            isSearchable={true} isClearable={true}
                                            options={authorities}
                                        // value={}
                                            name=""
                                            onChange={this.selectAuthority}/>
                                }


                                <Anchor onClick={this.signupAuthority}>
                                    {signupAuthority ? "I want to select an existing institution" : "My institution is not on the list" }
                                </Anchor>
                            </Box>


                        <Box border={{side:'bottom',color:'brand',size:'small'}}>
                            <SignupUser />


                            <Box direction="row" pad={{vertical:'medium'}} justify="between" align="center">
                                <Link to="/migrate">Migrate my PlanetLab Europe account</Link>
                                <Button type="submit" primary label="Signup" />
                            </Box>
                        </Box>
                        <Box direction="row" >
                            <Link to="/">Go back</Link>
                        </Box>
                        {message && <Text color="status-critical">{message}</Text>}
                    </Box>
                </Form>
            </Box>
        )
    }
}

Signup.contextType = EdgenetContext;

export default Signup;