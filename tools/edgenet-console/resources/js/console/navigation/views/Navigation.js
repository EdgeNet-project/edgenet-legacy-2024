import React from "react";
import {Grid, Box } from "grommet";

import Panel from "./Panel";
import Logo from "../components/Logo";
import { AuthorityMenu, MainMenu, ProfileMenu, RequestsMenu } from "../menu";

const Navigation = ({children}) => {

    return (
        <Grid rows={["100vh"]} columns={['small', 'flex']}
              areas={[{name: 'nav', start: [0, 0], end: [0, 0]}, {name: 'main', start: [1, 0], end: [1, 0]},]}>
            <Box gridArea="nav" background="light-1" fill>
                <Logo />
                <MainMenu />
                <AuthorityMenu />
                <RequestsMenu />
                <Box flex="grow"/>
                <Box pad={{vertical: 'medium'}}>
                    <ProfileMenu />
                </Box>
            </Box>
            <Box gridArea="main">
                <Panel>
                    {children}
                </Panel>
            </Box>
        </Grid>
    );
};

export default Navigation;