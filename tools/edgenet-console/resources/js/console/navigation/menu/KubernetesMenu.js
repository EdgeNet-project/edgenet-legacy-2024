import React from "react";
import {Dashboard, System} from "grommet-icons";
import NavigationButton from "../components/NavigationButton";
import NavigationSection from "../components/NavigationSection";

const ProfileMenu = () =>
    <NavigationSection label="Kubernetes">
        <NavigationButton label="Dashboard" path="/dashboard" icon={<Dashboard />} />
        <NavigationButton label="Configuration" path="/configuration" icon={<System />} />
    </NavigationSection>;
export default ProfileMenu;