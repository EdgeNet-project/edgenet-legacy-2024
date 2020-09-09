import React, { useContext } from "react";
import {Box, Heading, Text} from "grommet";
import { AuthenticationContext } from "../../authentication";

const Kubernetes = () => {
    const { user, edgenet } = useContext(AuthenticationContext);

    return (
        <Box pad="medium">
            <Heading size="small" margin="none">

            </Heading>

            <Box>
                Download Kubernetes config file
            </Box>

        </Box>
    );
}

export default Kubernetes;