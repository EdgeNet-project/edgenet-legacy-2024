import React from "react";
import {Route, Switch} from "react-router-dom";
import { Login, ForgotPassword, ResetPasswordView } from "./views";
import { Guest } from "./access";

const AuthenticationRoutes = () =>
    <Guest>
        <Switch>
            <Route exact path="/password/reset">
                <ForgotPassword />
            </Route>
            <Route path="/password/reset/:token">
                <ResetPasswordView />
            </Route>
            <Route exact path="/">
                <Login />
            </Route>
        </Switch>
    </Guest>;

export default AuthenticationRoutes;