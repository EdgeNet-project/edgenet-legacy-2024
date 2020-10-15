import React from "react";
import {Route, Switch} from "react-router-dom";
import Configuration from "../views/kubernetes/Configuration";
import Dashboard from "../views/kubernetes/Dashboard";

const KubernetesRoutes = () =>
    <Switch>
        <Route exact path="/kubernetes/configuration">
            <Configuration />
        </Route>
        <Route exact path="/kubernetes/dashboard">
            <Dashboard />
        </Route>
    </Switch>;

export default KubernetesRoutes;