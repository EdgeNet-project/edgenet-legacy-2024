import React, { useContext } from 'react';
import {Box, Button} from "grommet";
import axios from "axios";
import { AuthContext } from "../../auth";

import Header from "./Header";
// import Loading from "./Loading";
// import Footer from "./Footer";
import AUPText from "./AUPText";

const AUP = () => {
    const { user, edgenet_api, getEdgenetUser } = useContext(AuthContext)

    const acceptAUP = () => {
        console.log('accept')
        axios.patch(
            edgenet_api + '/apis/apps.edgenet.io/v1alpha/namespaces/authority-'+user.authority+'/acceptableusepolicies/' + user.name,
            [{ op: 'replace', path: '/spec/accepted', value: true }],
            { headers: { 'Content-Type': 'application/json-patch+json' } }
        )
            .then(res => {
                getEdgenetUser();
                console.log(res)
            })
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