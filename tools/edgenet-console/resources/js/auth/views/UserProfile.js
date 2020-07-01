import React, { useContext } from "react";
import {Box, Heading, Text} from "grommet";
import { AuthContext } from "../AuthContext";

const UserProfile = () => {
    const { user, edgenet } = useContext(AuthContext);

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