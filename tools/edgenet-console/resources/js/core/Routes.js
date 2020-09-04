import React from "react";
import {Grommet} from "grommet";
import {BrowserRouter as Router, Route, Redirect, Switch} from "react-router-dom";

import {AuthProvider} from "../auth";
import { Authenticated, Guest } from "../auth/access";
import { UserProfile } from "../auth/views";

import {NavigationView} from "../nav/views";

import ResourceList from "./ResourceList";
// import ResourceForm from "./ResourceForm";
// import ResourceView from "./ResourceView";
//
// import {Related} from "../form";
import {ForgotPasswordView, LoginView, ResetPasswordView, Signup, VerifyEmail} from "../auth/views";
import {
    ProfileNavigation,
    PasswordForm
} from "../views/profile";

const ProfileRoutes = () =>
    <Switch>
        <Route path="/profile/password">
            <PasswordForm />
        </Route>
        <Route>
            <UserProfile />
        </Route>
    </Switch>

const Routes = ({menu, theme}) =>
    <Grommet full theme={theme}>
        <Router>
            <AuthProvider>
                <Authenticated>
                    <NavigationView menu={menu}>
                        <Switch>
                            <Route path="/profile">
                                <ProfileRoutes />
                            </Route>
                            {/*<Route path={['/:resource/new', '/:resource/:id/edit']}*/}
                            {/*       component={ResourceForm}/>*/}
                            <Route path={['/:resource/:id', '/:resource']}
                                   component={ResourceList} />

                            <Redirect to="/profile" />
                        </Switch>
                        <Switch>
                            <Route path="/profile">
                                <ProfileNavigation />
                            </Route>
                            {/*<Route path={'/:resource/:id/edit'} component={Related}/>*/}
                            {/*<Route path="/:resource/new" component={null}/>*/}
                            {/*<Route path={['/:resource/:id', '/:resource']} component={ResourceView}/>*/}
                        </Switch>
                    </NavigationView>
                </Authenticated>

                <Guest>
                    <Switch>
                        <Route exact path="/password/reset">
                            <ForgotPasswordView />
                        </Route>
                        <Route path="/password/reset/:token"
                               render={({match}) => <ResetPasswordView token={match.params.token}/>}/>
                        <Route path="/signup">
                           <Signup />
                        </Route>
                        <Route path="/verify/:namespace/:code" children={<VerifyEmail />}/>
                        <Route path="/">
                            <LoginView />
                        </Route>
                    </Switch>
                </Guest>
            </AuthProvider>
        </Router>
    </Grommet>;

export default Routes;