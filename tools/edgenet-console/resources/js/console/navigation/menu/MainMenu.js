import React from "react";
import { Server } from "grommet-icons";
import NavigationButton from "../components/NavigationButton";

const MainMenu = () => {

    return (
            <NavigationButton label="Nodes" path="/nodes" icon={<Server />} />
    );
}

export default MainMenu;