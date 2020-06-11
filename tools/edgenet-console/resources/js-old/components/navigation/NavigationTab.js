import React from 'react';
import { Box, Button } from "grommet";
import { withRouter, matchPath } from "react-router";

class NavigationTab extends React.Component {

    constructor(props) {
        super(props);
        this.state = { hover: false };

        this.onClick = this.onClick.bind(this);
    }

    onClick(event) {
        const { history, path } = this.props;
        event.preventDefault();
        history.push(path);
    };

    render() {
        const { label='Tab', icon, location, path, exact, strict } = this.props;
        const { hover } = this.state;

        const active = matchPath(location.pathname, { path, exact, strict });
        const background = active && active.isExact ? "brand" : hover ? "light-2" : null;

        return (
            <Box pad="small" round={{size:'xsmall',corner:'top'}} background={background}
                 style={{cursor: 'pointer'}}
                 onMouseEnter={() => this.setState({hover: true})}
                 onMouseLeave={() => this.setState({hover: false})}
                 onClick={this.onClick}
            >
                <Button label={label} plain icon={icon} />
            </Box>
        )
    }
}

export default withRouter(NavigationTab);
