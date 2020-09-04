import React, { useContext } from "react";
import {Box, Heading, Text} from "grommet";
import { AuthenticationContext } from "../AuthenticationContext";

const UserProfile = () => {
    const { user, edgenet } = useContext(AuthenticationContext);

    return (
        <Box pad="medium">
            <Heading size="small" margin="none">
                {edgenet.spec.firstname} {edgenet.spec.lastname}
            </Heading>
            <Text>
                {edgenet.spec.email}
            </Text>
            <Text>
                {edgenet.spec.bio}
            </Text>

        </Box>
    );
}

export default UserProfile;