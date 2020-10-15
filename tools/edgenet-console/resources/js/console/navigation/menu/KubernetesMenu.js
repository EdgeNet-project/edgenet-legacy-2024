import React from "react";
import {Dashboard, System} from "grommet-icons";
import NavigationButton from "../components/NavigationButton";
import NavigationSection from "../components/NavigationSection";

const ProfileMenu = () =>
    <NavigationSection label="Kubernetes">
        <NavigationButton label="Dashboard" path="/kubernetes/dashboard" icon={<Dashboard />} />
        <NavigationButton label="Configuration" path="/kubernetes/configuration" icon={<System />} />
    </NavigationSection>;
export default ProfileMenu;