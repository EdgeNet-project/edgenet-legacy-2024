import React, {useContext} from "react";
import {User} from "grommet-icons";
import NavigationButton from "../components/NavigationButton";
import {AuthenticationContext} from "../../authentication";

const ProfileMenu = () => {
    const { user } = useContext(AuthenticationContext);

    return (
            <NavigationButton label={user.firstname} path='/profile' icon={<User/>}/>
    );
}

export default ProfileMenu;