import React from "react";
import {Route, Switch} from "react-router-dom";
import Profile from "../views/user/Profile";
import Password from "../views/user/Password";
import Secrets from "../views/user/Secrets";

const ClusterAdminRoutes = () =>
    <Switch>
        <Route exact path="/user">
            <Profile />
        </Route>
        <Route exact path="/user/secrets">
            <Secrets />
        </Route>
        <Route exact path="/user/password">
            <Password />
        </Route>
    </Switch>;

export default ClusterAdminRoutes;