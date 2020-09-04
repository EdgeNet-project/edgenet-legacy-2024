import React, {useState} from "react";
// import {Box, Form, Text, Button, Anchor} from "grommet";
// import {Link} from "react-router-dom";
// import Select from 'react-select';
import axios from "axios";

// // import { EdgenetContext } from "../../edgenet";
// import SignupSucces from "./SignupSucces";
// import SignupUser from "./SignupUser";
// import SignupAuthority from "./SignuAuthority";
// import Loading from "./Loading";
// import Header from "./Header";
// import Footer from "./Footer";

const RegistrationContext = React.createContext({});
const RegistrationConsumer = RegistrationContext.Consumer;

const Registration = ({children}) => {
    const [ user, setUser ] = useState({
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
    });
    const [ authority, setAuthority ] = useState(null);
    const [ message, setMessage ] = useState(null);
    const [ error, setError ] = useState(null);
    const [ loading, setLoading ] = useState(null);


    console.log('authority => ', authority)


    const submitRegistration = ({value}) => {

        console.log(value)

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

    }

    const signupAuthority = () => {
        const { signupAuthority } = this.state;

        this.setState({
            signupAuthority: !signupAuthority,
            authority: null
        })
    }



    const selectAuthority = ({value}) => {
        this.setState({
            authority: value,
            signupAuthority: null
        })
    }


    //
    // if (success) {
    //     return <SignupSucces />;
    // }
    //
    // if (loading) {
    //     return <Loading />;
    // }

    return (
        <RegistrationContext.Provider value={{
            setAuthority: setAuthority,
            submitRegistration: submitRegistration
        }}>
            {children}
        </RegistrationContext.Provider>

    )

}

export {Registration, RegistrationContext, RegistrationConsumer };