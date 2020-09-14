import React, {useContext} from "react";
import {Grid, Box, Nav, Text } from "grommet";

import Panel from "./Panel";
import Logo from "../components/Logo";
import NavigationButton from "../components/NavigationButton";
import {User} from "grommet-icons";
import {ConsoleContext} from "../../index";
import {AuthenticationContext} from "../../authentication";

const NavigationSection = ({section}) =>
    <Nav gap="none">
        {section.label && <Box border={{side:'top',color:'light-4'}} pad={{horizontal:'medium', vertical:'small'}} margin={{top:'small'}}>
            <Text size="small">{section.label}</Text>
        </Box>}
        {section.menu.map((menu, k) =>
            <NavigationButton key={"menu-" + k} label={menu.label} path={menu.path} icon={menu.icon} />
        )}
    </Nav>;

const Navigation = ({children}) => {
    const { navigation } = useContext(ConsoleContext);
    const { user } = useContext(AuthenticationContext);

    return (
        <Grid rows={["100vh"]} columns={['small', 'flex']}
              areas={[{name: 'nav', start: [0, 0], end: [0, 0]}, {name: 'main', start: [1, 0], end: [1, 0]},]}>
            <Box gridArea="nav" background="light-1" fill>
                <Logo />
                { navigation.map((section, j) => <NavigationSection  key={"menu-section-" + j} section={section} />) }
                <Box flex="grow"/>
                <Box pad={{vertical: 'medium'}}>
                    <NavigationButton label={user.name} path='/profile' icon={<User/>}/>
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