import React, {useContext} from "react";
import {Box} from "grommet";
import {User} from "grommet-icons";
import NavigationButton from "../components/NavigationButton";
import {AuthenticationContext} from "../../authentication";

const ProfileMenu = () => {
    const { user } = useContext(AuthenticationContext);

    return (
        <Box border={{side:'top',color:'light-4'}} pad={{top:'small', bottom:'medium'}} margin={{top:'small'}}>
            <NavigationButton label={user.firstname} path="/user" icon={<User/>} />
        </Box>
    );
}

export default ProfileMenu;