import React, {useContext} from "react";
import {Box} from "grommet";
import {Dashboard, System, User} from "grommet-icons";
import NavigationButton from "../components/NavigationButton";
import {AuthenticationContext} from "../../authentication";

const ProfileMenu = () => {
    const { user } = useContext(AuthenticationContext);

    return (
        <Box>
            <NavigationButton label="Kubernetes" path="/kubernetes" icon={<Dashboard />} />
            <NavigationButton label="Configuration" path="/configuration" icon={<System />} />
            <NavigationButton label={user.firstname} path='/profile' icon={<User/>} />
        </Box>
    );
}

export default ProfileMenu;