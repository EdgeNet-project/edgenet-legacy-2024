import React from "react";
import {useHistory, useRouteMatch} from "react-router-dom";
import {Box, Button, Text} from "grommet";

const NavigationButton = ({label, path, icon, onClick}) => {
    const history = useHistory();
    const match = useRouteMatch(path);

    return (
        <Button plain alignSelf="stretch"
                onClick={() => history.push(path)} active={!!match} hoverIndicator="white">
            <Box pad={{vertical: "xsmall", horizontal: "medium"}}
                 gap="xxsmall" direction="row" background={match ? {color:"neutral-2", dark:true} : null}>{icon} <Text>{label}</Text></Box>
        </Button>
    );
}

export default NavigationButton;