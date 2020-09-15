import React from "react";
import { Organization, User } from "grommet-icons";
import { Admin } from "../../authentication/access";
import NavigationButton from "../components/NavigationButton";
import NavigationSection from "../components/NavigationSection";

const RequestsMenu = () => {

    return (
        <Admin>
            <NavigationSection label="Requests">
                <NavigationButton label="Users" path="/userrequests" icon={<User />} />
                <NavigationButton label="Authorities" path="/authorityrequests" icon={<Organization />} />
            </NavigationSection>
        </Admin>
    );
}

export default RequestsMenu;