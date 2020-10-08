import React from "react";
import { Organization, User } from "grommet-icons";
import { Admin, ClusterAdmin } from "../../authentication/access";
import NavigationButton from "../components/NavigationButton";
import NavigationSection from "../components/NavigationSection";

const RequestsMenu = () => {

    return (
        <NavigationSection label="Requests">
            <Admin>
                <NavigationButton label="Users" path="/userrequests" icon={<User />} />
            </Admin>
            <ClusterAdmin>
                <NavigationButton label="Authorities" path="/authorityrequests" icon={<Organization />} />
            </ClusterAdmin>
        </NavigationSection>
    );
}

export default RequestsMenu;