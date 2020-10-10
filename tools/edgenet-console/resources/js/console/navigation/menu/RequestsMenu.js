import React from "react";
import { Organization, User } from "grommet-icons";
import { Admin, ClusterAdmin } from "../../authentication/access";
import NavigationButton from "../components/NavigationButton";
import NavigationSection from "../components/NavigationSection";

const RequestsMenu = () => {

    return (
        <Admin>
        <NavigationSection label="Requests">
            <NavigationButton label="Users" path="/userrequests" icon={<User />} />
            <ClusterAdmin>
                <NavigationButton label="Authorities" path="/authorityrequests" icon={<Organization />} />
            </ClusterAdmin>
        </NavigationSection>
        </Admin>
    );
}

export default RequestsMenu;