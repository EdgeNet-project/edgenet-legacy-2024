import React from "react";
import {Route, Switch} from "react-router-dom";
import AuthorityList from "../views/requests/AuthorityList";
import UserList from "../views/requests/UserList";

const RequestRoutes = () =>
    <Switch>
        <Route exact path="/requests/authorities">
            <AuthorityList />
        </Route>
        <Route exact path="/requests/users">
            <UserList />
        </Route>
    </Switch>;

export default RequestRoutes;