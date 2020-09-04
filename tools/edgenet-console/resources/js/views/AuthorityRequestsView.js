import React from "react";
import { Box } from "grommet";

const AuthorityRequestsView = ({item}) =>
    <Box pad="medium">
        {item.title}

        <p>
        {item.text}
        </p>
    </Box>;

export default AuthorityRequestsView;
