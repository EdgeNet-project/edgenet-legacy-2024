import React, { useState, useEffect } from 'react';
import {Link} from "react-router-dom";
import { useParams } from "react-router";
import {Box} from "grommet";
import axios from "axios";

import {ConsoleLogo} from "../../index";


const Loading = () =>
    <Box align="center">
        <Box gap="medium" alignSelf="center" width="medium" alignContent="center" align="stretch">
            <Box margin={{vertical:'medium'}}>
                <ConsoleLogo />
            </Box>
            <Box pad={{vertical:'medium'}}>
                Please wait...
            </Box>
        </Box>
    </Box>

const VerifyEmail = () => {
    const [ verified, setVerified ] = useState(false);
    const { namespace, code } = useParams();

    useEffect(() => {
        axios.patch(
            '/apis/apps.edgenet.io/v1alpha/namespaces/' + namespace + '/emailverifications/' + code,
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
        return (
            <Box align="center">
                <Box gap="medium" alignSelf="center" width="medium" alignContent="center" align="stretch">
                    <Box margin={{vertical:'medium'}}>
                        <ConsoleLogo />
                    </Box>
                    <Box pad={{vertical:'medium'}}>
                        Verifying email address, please wait...
                    </Box>
                </Box>
            </Box>
        )
    } else {
        return (
            <Box align="center">
                <Box gap="medium" alignSelf="center" width="medium" alignContent="center" align="stretch">
                    <Box margin={{vertical:'medium'}}>
                        <ConsoleLogo />
                        E-Mail verification
                    </Box>
                    <Box pad={{vertical:'medium'}}>
                        Your email has been verified, Thank you!<br/>
                        We will review your information and come back to you shortly!
                    </Box>
                    <Box direction="row" >
                        <Link to="/">Go back to the login page</Link>
                    </Box>
                </Box>
            </Box>
        );
    }

}

export default VerifyEmail;