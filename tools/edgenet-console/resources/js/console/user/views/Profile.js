import React, { useContext } from "react";
import {Box, Heading, Text} from "grommet";
import { AuthenticationContext } from "../../authentication";

const Profile = () => {
    const { user, edgenet } = useContext(AuthenticationContext);

    return (
        <Box pad="medium">
            <Heading size="small" margin="none">
                {user.firstname} {user.lastname}
            </Heading>
            <Text>
                {user.email}
            </Text>
            <Text>
            </Text>

        </Box>
    );
}

export default Profile;