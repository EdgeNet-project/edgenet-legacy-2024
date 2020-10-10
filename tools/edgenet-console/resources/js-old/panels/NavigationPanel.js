import React from 'react';
import { Grid, Box } from "grommet";
import { User, Organization, Workshop  } from "grommet-icons";
import { NavigationMenu } from '../components/navigation';
import { UserConsumer } from "../components/user";

const NavigationPanel = ({children}) =>
    <Grid rows={["100vh"]} columns={['small', 'flex']}
          areas={[{name: 'nav', start: [0, 0], end: [0, 0]}, {name: 'main', start: [1, 0], end: [1, 0]},]}>
        <Box gridArea="nav" background="brand" fill>
            <Box pad="medium" align="start">
                EdgeNet
            </Box>

            <NavigationMenu label="Stages" path="/etudiants/stages" icon={<Workshop />} />

            <Box flex="grow" />
            <Box pad={{vertical: 'medium'}}>
                <UserConsumer>
                    {({user}) => <NavigationMenu label={user.firstname} path='/pro/profile' icon={<User/>}/>}
                </UserConsumer>
            </Box>
            {/*{menu && <NavigationLayer setClose={() => setMenu(false)} />}*/}
        </Box>
        <Box gridArea="main">
            {children}
        </Box>
    </Grid>;


export default NavigationPanel;