import React from "react";
import {Grid, Box } from "grommet";

import Panel from "./Panel";
import Logo from "../components/Logo";
import { MyAuthorityMenu, MainMenu, ProfileMenu, RequestsMenu, KubernetesMenu, ClusterAdminMenu } from "../menu";

const Navigation = ({children}) => {

    return (
        <Grid rows={["100vh"]} columns={['small', 'flex']}
              areas={[{name: 'nav', start: [0, 0], end: [0, 0]}, {name: 'main', start: [1, 0], end: [1, 0]},]}>
            <Box gridArea="nav" background="#F8FAFE" fill>
                <Logo />
                <MainMenu />
                <MyAuthorityMenu />
                <RequestsMenu />
                <Box flex="grow"/>
                <ClusterAdminMenu />
                <KubernetesMenu />
                <ProfileMenu />
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