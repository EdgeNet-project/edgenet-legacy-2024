import React from "react";
import {Route, Switch} from "react-router-dom";
import { Login, ForgotPassword, ResetPasswordView } from "../authentication";
import { Guest } from "../authentication/access";
import { UserRegistration, VerifyEmail } from "../registration";

const ConsoleRoutes = () =>
    <Guest>
        <Switch>
            <Route path="/signup">
                <UserRegistration />
            </Route>
            <Route path="/verify/:namespace/:code">
                <VerifyEmail />
            </Route>
            <Route exact path="/password/reset">
                <ForgotPassword />
            </Route>
            <Route path="/password/reset/:token">
                <ResetPasswordView />
            </Route>
            <Route>
                <Login />
            </Route>
        </Switch>
    </Guest>;

export default ConsoleRoutes;