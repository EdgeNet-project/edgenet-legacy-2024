import React from "react";
import {Route, Switch} from "react-router-dom";
import NodeList from "../views/myauthority/NodeList";
import SliceList from "../views/myauthority/SliceList";
import UserList from "../views/myauthority/UserList";

const MyAuthorityRoutes = () =>
    <Switch>
        <Route exact path="/myauthority/nodes">
            <NodeList />
        </Route>
        <Route exact path="/myauthority/slices">
            <SliceList />
        </Route>
        <Route exact path="/myauthority/users">
            <UserList />
        </Route>
    </Switch>;

export default MyAuthorityRoutes;