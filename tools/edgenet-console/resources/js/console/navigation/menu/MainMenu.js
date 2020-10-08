import React from "react";
import { Box } from "grommet";
import { Server } from "grommet-icons";
import NavigationButton from "../components/NavigationButton";

const MainMenu = () => {

    return (
        <Box>
            <NavigationButton label="Nodes" path="/nodes" icon={<Server />} />
            <NavigationButton label="Slices" path="/slices" icon={<Server />} />
        </Box>
    );
}

export default MainMenu;