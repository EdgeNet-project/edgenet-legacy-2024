import React from "react";
import {Route, Switch} from "react-router-dom";
import { Login, ForgotPassword } from "./views";
import { Guest } from "./access";

const AuthenticationRoutes = () =>
    <Guest>
        <Switch>
            <Route exact path="/password/reset">
                <ForgotPassword />
            </Route>
            {/*<Route path="/password/reset/:token"*/}
            {/*       render={({match}) => <ResetPasswordView token={match.params.token}/>}/>*/}
            {/*<Route path="/signup">*/}
            {/*    <Signup />*/}
            {/*</Route>*/}
            {/*<Route path="/verify/:namespace/:code" children={<VerifyEmail />}/>*/}
            <Route exact path="/">
                <Login />
            </Route>
        </Switch>
    </Guest>;

export default AuthenticationRoutes;