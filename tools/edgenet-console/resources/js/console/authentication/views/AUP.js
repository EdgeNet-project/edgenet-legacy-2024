import React, { useContext } from 'react';
import {Box, Button} from "grommet";
import axios from "axios";
import { AuthenticationContext } from "../AuthenticationContext";

import Header from "./Header";
import AUPText from "./AUPText";

const AUP = () => {
    const { user, getAUP } = useContext(AuthenticationContext)

    const acceptAUP = () => {
        axios.patch(
            '/apis/apps.edgenet.io/v1alpha/namespaces/authority-'+user.authority+'/acceptableusepolicies/' + user.name,
            [{ op: 'replace', path: '/spec/accepted', value: true }],
            { headers: { 'Content-Type': 'application/json-patch+json' } }
        )
            .then(res => getAUP())
            .catch(err => console.log(err.message))
    }

    // if (!verified) {
    //     return <Loading title="E-Mail verification" />
    // } else {
        return (
            <Box align="center">
                <Header title="EdgeNet Acceptable Use Policy (AUP)" />
                <Box margin={{vertical:'medium'}} width="large" height="60vh">
                    <AUPText />
                </Box>
                <Button primary label="Accept" onClick={acceptAUP} />
                {/*<Footer />*/}
            </Box>
        );
    // }

}

export default AUP;