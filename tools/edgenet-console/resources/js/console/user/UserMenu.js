import React from "react";
import { Box } from "grommet";
import { Lock, System } from "grommet-icons";
import { Button } from "../navigation";

const UserMenu = () =>
    <Box pad={{vertical:'medium'}}>
        <Button label="Kubernetes" path="/profile/kubernetes" icon={<System />} />
        <Button label="Update Password" path="/profile/password" icon={<Lock />} />
    </Box>

export default UserMenu;