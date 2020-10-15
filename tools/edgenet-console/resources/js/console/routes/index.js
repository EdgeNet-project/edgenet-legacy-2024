import React from "react";
import {Route, Redirect, Switch} from "react-router-dom";
import { Login, ForgotPassword, ResetPasswordView } from "../authentication";
import { Guest } from "../authentication/access";
import { UserRegistration, VerifyEmail } from "../registration";

import ClusterAdminRoutes from "./ClusterAdminRoutes";
import KubernetesRoutes from "./KubernetesRoutes";
import MyAuthorityRoutes from "./MyAuthorityRoutes";
import RequestsRoutes from "./RequestsRoutes";
import UserRoutes from "./UserRoutes";

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
            <Route exact path="/">
                <Login />
            </Route>
            <Redirect to="/" />
        </Switch>
    </Guest>;

export {
    ConsoleRoutes, ClusterAdminRoutes, KubernetesRoutes, MyAuthorityRoutes, RequestsRoutes, UserRoutes

};