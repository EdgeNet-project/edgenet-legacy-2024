import React from "react";
import {Route} from "react-router-dom";

import { Registration } from "./RegistrationContext";

import UserRegistration from "./views/UserRegistration";
import VerifyEmail from "./views/VerifyEmail";

const RegistrationRoutes = () =>
    <Registration>
        <Route path="/signup">
            <UserRegistration />
        </Route>
        <Route path="/verify/:namespace/:code">
            <VerifyEmail />
        </Route>
    </Registration>;

export default RegistrationRoutes;