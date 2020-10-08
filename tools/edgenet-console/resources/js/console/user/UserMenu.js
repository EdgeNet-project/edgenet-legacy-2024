import React, {useContext} from "react";
import {Box, Button as GrommetButton, Text} from "grommet";
import { Lock, License, System, Logout } from "grommet-icons";
import { NavigationButton } from "../navigation";
import {AuthenticationContext} from "../authentication";

const UserMenu = () => {
    const { logout } = useContext(AuthenticationContext);

    return (
        <Box pad={{vertical:'medium'}}>
            <NavigationButton label="Secrets" path="/profile/secrets" icon={<License />} />
            <NavigationButton label="Update Password" path="/profile/password" icon={<Lock />} />
            <Box border={{side:'top'}} pad={{top:'medium'}} margin={{top:'medium'}}>
                <GrommetButton plain alignSelf="stretch"
                               onClick={logout} hoverIndicator="white">
                    <Box pad={{vertical: "xsmall", horizontal: "medium"}}
                         gap="xxsmall" direction="row" ><Logout /> <Text>Logout</Text></Box>
                </GrommetButton>
            </Box>
        </Box>
    )
}



export default UserMenu;