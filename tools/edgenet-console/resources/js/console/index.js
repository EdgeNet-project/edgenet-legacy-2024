import React, {useContext} from "react";
import {BrowserRouter as Router, Redirect, Route, Switch} from "react-router-dom";
import {Grommet, Image, Text} from "grommet";
import theme from "./theme";

import { Authentication, AuthenticationRoutes } from "./authentication";
import { RegistrationRoutes } from "./registration";

const ConsoleContext = React.createContext({
    logo: null
});
const ConsoleConsumer = ConsoleContext.Consumer;

const ConsoleLogo = () => {
    const { title, logo } = useContext(ConsoleContext);

    return logo ? <Image fill src={logo} alt={title || document.title} /> : <Text>{title || document.title}</Text>;
};

const Console = ({
                     title,
                     logo,
                     menu
}) =>
    <ConsoleContext.Provider value={{
        title: title,
        logo: logo
    }}>
        <Grommet full theme={theme}>
            <Router>
                <Authentication>
                    <AuthenticationRoutes />
                </Authentication>

                <RegistrationRoutes />
            </Router>
        </Grommet>
    </ConsoleContext.Provider>;

export { Console, ConsoleContext, ConsoleConsumer, ConsoleLogo };