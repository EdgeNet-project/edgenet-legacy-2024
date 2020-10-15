import React from "react";
import { Organization, User } from "grommet-icons";
import { ClusterAdmin } from "../../authentication/access";
import NavigationButton from "../components/NavigationButton";
import NavigationSection from "../components/NavigationSection";

const AdminMenu = () => {

    return (
        <ClusterAdmin>
            <NavigationSection label="Admin">
                <NavigationButton label="Authorities" path="/admin/authorities" icon={<Organization />} />
                <NavigationButton label="Users" path="/admin/users" icon={<User />} />
            </NavigationSection>
        </ClusterAdmin>
    );
}

export default AdminMenu;