import React, {useContext} from "react";
import {Box} from "grommet";
import {System, User} from "grommet-icons";
import NavigationButton from "../components/NavigationButton";
import {AuthenticationContext} from "../../authentication";

const ProfileMenu = () => {
    const { user } = useContext(AuthenticationContext);

    return (
        <Box>
            <NavigationButton label={user.firstname} path='/profile' icon={<User/>} />
            <NavigationButton label="Configuration" path="/configuration" icon={<System />} />
        </Box>
    );
}

export default ProfileMenu;