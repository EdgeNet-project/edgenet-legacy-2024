import React, { useState, useEffect, useContext } from 'react';
import { useParams, useHistory } from "react-router";
import {Box} from "grommet";
import axios from "axios";
import { EdgenetContext } from "../../edgenet";

import Header from "./Header";

const VerifyEmail = () => {
    const { api } = useContext(EdgenetContext)
    const [ verified, setVerified ] = useState(false);
    const { namespace, code } = useParams();
    const history = useHistory();

    useEffect(() => {
        axios.patch(
            api + '/apis/apps.edgenet.io/v1alpha/namespaces/' + namespace + '/emailverifications/' + code,
            [{ op: 'replace', path: '/spec/verified', value: true }],
            { headers: { 'Content-Type': 'application/json-patch+json' } }
        )
            .then(res => console.log(res))
            .catch(err => console.log(err.message))
    })

    return (
        <Box align="center">
            <Header title="E-Mail verification" />
            Please wait...
        </Box>
    );
}

export default VerifyEmail;