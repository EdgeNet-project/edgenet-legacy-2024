import React, {useContext} from "react";
import {BrowserRouter as Router, Route, Switch} from "react-router-dom";
import {Grommet, Image, Text} from "grommet";
import theme from "./theme";

import ConsoleRoutes from "./routes";
import { Authentication } from "./authentication";
import { Authenticated } from "./authentication/access"
import { Navigation } from "./navigation";

import Nodes from "./resources/views/Nodes";
import Slices from "./resources/views/Slices";
import AuthorityRequests from "./resources/views/AuthorityRequests";
import UserRequests from "./resources/views/UserRequests";

import Profile from "./user/views/Profile";
import Configuration from "./user/views/Configuration";
import Secrets from "./user/views/Secrets";
import Dashboard from "./user/views/Dashboard";
import Password from "./user/views/Password";

import UserMenu from "./user/UserMenu";

const ConsoleContext = React.createContext({
    logo: null
});
const ConsoleConsumer = ConsoleContext.Consumer;

const ConsoleLogo = () => {
    const { title, logo } = useContext(ConsoleContext);

    return logo ? <Image fill src={logo} alt={title || document.title} /> : <Text>{title || document.title}</Text>;
};

const Console = ({settings}) =>
    <ConsoleContext.Provider value={{
        title: settings.title,
        logo: settings.logo,
        navigation: settings.navigation
    }}>
        <Grommet full theme={theme}>
            <Router>
                <Authentication>

                    <Authenticated>
                        <Navigation>
                            <Switch>
                                <Route exact path="/nodes">
                                    <Nodes />
                                </Route>
                                <Route exact path="/slices">
                                    <Slices />
                                </Route>

                                <Route exact path="/authorityrequests">
                                    <AuthorityRequests />
                                </Route>
                                <Route exact path="/userrequests">
                                    <UserRequests />
                                </Route>

                                <Route exact path="/profile">
                                    <Profile />
                                </Route>
                                <Route exact path="/configuration">
                                    <Configuration />
                                </Route>
                                <Route exact path="/dashboard">
                                    <Dashboard />
                                </Route>

                                <Route exact path="/profile/secrets">
                                    <Secrets />
                                </Route>
                                <Route exact path="/profile/password">
                                    <Password />
                                </Route>
                            </Switch>


                            <Switch>
                                <Route path="/profile">
                                    <UserMenu />
                                </Route>
                            </Switch>

                        </Navigation>
                    </Authenticated>

                    <ConsoleRoutes />

                </Authentication>
            </Router>
        </Grommet>
    </ConsoleContext.Provider>;

export { Console, ConsoleContext, ConsoleConsumer, ConsoleLogo };