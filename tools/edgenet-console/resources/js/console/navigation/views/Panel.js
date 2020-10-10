import React from "react";
import { Grid, Box } from "grommet";

const Panel = ({children}) => {
    const count = React.Children.count(children);

    if (count === 2) {
        children = React.Children.toArray(children);
        return (
            <Grid fill rows={['auto']} columns={['flex', 'medium']}
                  areas={[{name: 'main', start: [0, 0], end: [0, 0]}, {name: 'side', start: [1, 0], end: [1, 0]}]}>
                <Box gridArea="main" overflow="auto">{children[0]}</Box>
                <Box gridArea="side" background="light-1" fill overflow="auto">{children[1]}</Box>
            </Grid>
        );
    }

    return <Box fill>{children}</Box>;
};

export default Panel;