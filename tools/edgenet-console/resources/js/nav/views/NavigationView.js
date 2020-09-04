import React from "react";

import { Grid, Box } from "grommet";
import { User } from "grommet-icons";

import { AuthConsumer } from "../../auth";
import { NavMenu } from "../components";

const NavigationPanel = ({children}) => {
    const count = React.Children.count(children);

    if (count === 2) {
        children = React.Children.toArray(children);
        return (
            <Grid fill rows={['auto']} columns={['flex', 'medium']}
                  areas={[{name: 'main', start: [0, 0], end: [0, 0]}, {name: 'side', start: [1, 0], end: [1, 0]}]}>
                <Box gridArea="main">{children[0]}</Box>
                <Box gridArea="side" background="light-1" fill overflow="auto">{children[1]}</Box>
            </Grid>
        );
    }

    return <Box fill>{children}</Box>;
};

const NavigationView = ({children, logo, title, menu = []}) =>
    <Grid rows={["100vh"]} columns={['small', 'flex']}
          areas={[{name: 'nav', start: [0, 0], end: [0, 0]}, {name: 'main', start: [1, 0], end: [1, 0]},]}>
        <Box gridArea="nav" background="brand" fill>
            <Box pad="medium" align="start">
            {title ? title: document.title}
            </Box>
            {
                menu.filter(m => m.main).map((m, k) => <NavMenu key={"menu-"+k} label={m.label} path={m.path} icon={m.icon} />)
            }
            <Box flex="grow" />
            <Box pad={{vertical: 'medium'}}>
                {
                    menu.filter(m => !m.main).map((m, k) => <NavMenu key={"menu-"+k} label={m.label} path={m.path} icon={m.icon} />)
                }
                <AuthConsumer>
                    {({edgenet}) => <NavMenu label={edgenet.spec.firstname} path='/profile' icon={<User/>} />}
                </AuthConsumer>
            </Box>
            {/*{menu && <NavigationLayer setClose={() => setMenu(false)} />}*/}
        </Box>
        <Box gridArea="main">
            <NavigationPanel>
                {children}
            </NavigationPanel>
        </Box>
    </Grid>;

export default NavigationView;
