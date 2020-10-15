import React from "react";
import {Route, Switch} from "react-router-dom";
import AuthorityList from "../views/admin/AuthorityList";
import UserList from "../views/admin/UserList";
import SliceList from "../views/admin/SliceList";

const ClusterAdminRoutes = () =>
    <Switch>
        <Route exact path="/admin/authorities">
            <AuthorityList />
        </Route>
        <Route exact path="/admin/users">
            <UserList />
        </Route>
        <Route exact path="/admin/slices">
            <SliceList />
        </Route>
    </Switch>;

export default ClusterAdminRoutes;