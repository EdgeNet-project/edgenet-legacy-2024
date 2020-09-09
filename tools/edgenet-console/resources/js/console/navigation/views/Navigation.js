import React, {useContext} from "react";
import {Box, Grid} from "grommet";

import Panel from "./Panel";
import Logo from "../components/Logo";
import Button from "../components/Button";
import {User} from "grommet-icons";
import {ConsoleContext} from "../../index";
import {AuthenticationContext} from "../../authentication";

const Navigation = ({children}) => {
    const { navigation } = useContext(ConsoleContext);
    const { user } = useContext(AuthenticationContext);

    return (
        <Grid rows={["100vh"]} columns={['small', 'flex']}
              areas={[{name: 'nav', start: [0, 0], end: [0, 0]}, {name: 'main', start: [1, 0], end: [1, 0]},]}>
            <Box gridArea="nav" background="light-1" fill>
                <Logo />
                { navigation.map(resource => <Button key={"menu-" + resource.name} label={resource.label} path={resource.path} icon={resource.icon} />) }
                <Box flex="grow"/>
                <Box pad={{vertical: 'medium'}}>
                    <Button label={user.name} path='/profile' icon={<User/>}/>
                </Box>
                {/*{menu && <NavigationLayer setClose={() => setMenu(false)} />}*/}
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