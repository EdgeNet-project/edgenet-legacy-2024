import React from "react";
import { Box } from "grommet";
import { Lock } from "grommet-icons";
import { NavMenu } from "../../nav/components";

const ProfileNavigation = () =>
    <Box pad={{vertical:'medium',horizontal:'small'}}>
        <NavMenu label="Update Passoword" path="/profile/password" icon={<Lock />} />
    </Box>

export default ProfileNavigation;