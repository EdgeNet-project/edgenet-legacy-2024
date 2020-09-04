import React, { useState, useEffect, useContext } from 'react';
import { useParams, useHistory } from "react-router";
import {Box} from "grommet";
import axios from "axios";
// import { EdgenetContext } from "../../edgenet";

import Header from "./Header";
import Loading from "./Loading";
import Footer from "./Footer";

const VerifyEmail = () => {
    // const { api } = useContext(EdgenetContext)
    const [ verified, setVerified ] = useState(false);
    const { namespace, code } = useParams();
    const history = useHistory();

    useEffect(() => {
        axios.patch(
            api + '/apis/apps.edgenet.io/v1alpha/namespaces/' + namespace + '/emailverifications/' + code,
            [{ op: 'replace', path: '/spec/verified', value: true }],
            { headers: { 'Content-Type': 'application/json-patch+json' } }
        )
            .then(res => {
                setVerified(true)
                console.log(res)
            })
            .catch(err => console.log(err.message))
    })

    if (!verified) {
        return <Loading title="E-Mail verification" />
    } else {
        return (
            <Box align="center">
                <Header title="E-Mail verification" />
                <Box pad={{vertical:'medium'}}>
                    Your email has been verified, Thank you!<br/>
                    We will review your information and come back to you shortly!
                </Box>
                <Footer />
            </Box>
        );
    }

}

export default VerifyEmail;