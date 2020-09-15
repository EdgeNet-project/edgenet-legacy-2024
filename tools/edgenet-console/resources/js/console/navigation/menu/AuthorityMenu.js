import React from "react";
import { Cluster, User, Server } from "grommet-icons";
import { Admin } from "../../authentication/access";
import NavigationButton from "../components/NavigationButton";
import NavigationSection from "../components/NavigationSection";

const AuthorityMenu = () => {

    return (
        <Admin>
            <NavigationSection label="My Authority">
                <NavigationButton label="Nodes" path="/nodes" icon={<Server />} />
                <NavigationButton label="Slices" path="/slices" icon={<Cluster />} />
                <NavigationButton label="Users" path="/users" icon={<User />} />
            </NavigationSection>
        </Admin>
    );
}

export default AuthorityMenu;