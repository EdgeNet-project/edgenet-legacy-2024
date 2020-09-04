import React from "react";
import {Route} from "react-router-dom";
import { Registration } from "./RegistrationContext";
import { UserRegistration } from "./views";

const RegistrationRoutes = () =>
    <Registration>
        <Route path="/signup">
            <UserRegistration />
        </Route>
    </Registration>;

export default RegistrationRoutes;