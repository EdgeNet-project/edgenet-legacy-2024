import React, {useContext} from "react";
import {BrowserRouter as Router, Route, Switch} from "react-router-dom";
import {Grommet, Image, Text} from "grommet";
import theme from "./theme";

import {
    ConsoleRoutes,
    ClusterAdminRoutes,
    MyAuthorityRoutes,
    KubernetesRoutes,
    RequestsRoutes,
    UserRoutes
} from "./routes";

import UserMenu from "./navigation/secondary/user/UserMenu";

import { Authentication } from "./authentication";
import { Authenticated } from "./authentication/access"
import { Navigation } from "./navigation";

import NodeList from "./views/main/NodeList";

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
                                    <NodeList />
                                </Route>

                                <Route path="/myauthority">
                                    <MyAuthorityRoutes />
                                </Route>
                                <Route path="/requests">
                                    <RequestsRoutes />
                                </Route>
                                <Route path="/admin">
                                    <ClusterAdminRoutes />
                                </Route>
                                <Route path="/kubernetes">
                                    <KubernetesRoutes />
                                </Route>
                                <Route path="/user">
                                    <UserRoutes />
                                </Route>

                            </Switch>


                            <Switch>
                                <Route path="/user">
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