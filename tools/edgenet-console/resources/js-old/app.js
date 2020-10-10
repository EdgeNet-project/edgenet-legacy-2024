import React from "react";
import ReactDOM from "react-dom";
import { BrowserRouter as Router, Route, Switch, Redirect } from "react-router-dom";
import { Grommet, Box } from "grommet";
import theme from "./theme";

import { UserProvider } from "./components/user";
import { Authenticated, Anonymous } from "./components/user/access";
import { LoginForm } from "./panels/login";
import NavigationPanel from "./panels/NavigationPanel";
import RegistrationPanel from "./panels/RegistrationPanel";
import EmailVerificationPanel from "./panels/EmailVerificationPanel";


const EdgenetConsole = () =>
    <Grommet theme={theme}>
        <Router>
            <UserProvider>
                <Authenticated>
                    <NavigationPanel>
                            <Switch>
                                <Redirect exact path="/" to="/user/profile" />
                            </Switch>
                    </NavigationPanel>
                </Authenticated>

                <Anonymous>
                    <Switch>
                        <Route path="/signup" component={RegistrationPanel} />
                        <Route path="/verify/:token?" component={EmailVerificationPanel} />
                        <Route component={LoginForm} />
                    </Switch>
                </Anonymous>

            </UserProvider>
        </Router>
    </Grommet>;

const dom = document.getElementById('edgenet-console');
if (dom) {
    ReactDOM.render(<EdgenetConsole />, dom);
}
