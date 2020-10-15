import React from "react";
import { Cluster, User, Server } from "grommet-icons";
import { Admin } from "../../authentication/access";
import NavigationButton from "../components/NavigationButton";
import NavigationSection from "../components/NavigationSection";

const MyAuthorityMenu = () => {
    return (
        <Admin>
            <NavigationSection label="My Authority">
                <NavigationButton label="Nodes" path="/myauthority/nodes" icon={<Server />} />
                <NavigationButton label="Slices" path="/myauthority/slices" icon={<Cluster />} />
                <NavigationButton label="Users" path="/myauthority/users" icon={<User />} />
            </NavigationSection>
        </Admin>
    );
}

export default MyAuthorityMenu;