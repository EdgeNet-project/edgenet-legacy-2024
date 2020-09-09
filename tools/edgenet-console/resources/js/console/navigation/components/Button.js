import React from "react";
import {useHistory, useRouteMatch} from "react-router-dom";
import {Box, Button as GrommetButton, Text} from "grommet";

const Button = ({label, path, icon}) => {
    const history = useHistory();
    const match = useRouteMatch(path);

    const background = match ? "white" : null;

    return (
        <GrommetButton plain alignSelf="stretch"
                onClick={() => history.push(path)} active={!!match} hoverIndicator="white">
            <Box pad={{vertical: "xsmall", horizontal: "medium"}}
                 gap="xxsmall" direction="row" background={background}>{icon} <Text>{label}</Text></Box>
        </GrommetButton>
    );
}

export default Button;