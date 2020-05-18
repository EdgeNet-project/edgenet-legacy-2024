import React from "react";
import PropTypes from "prop-types";
import { matchPath, withRouter } from "react-router";
import { Box, Button } from "grommet";

class NavMenu extends React.Component {

    constructor(props) {
        super(props);
        this.state = {
            hover: false
        };
        this.onClick = this.onClick.bind(this);
    }

    onClick(event) {
        event.preventDefault();
        this.props.history.push(this.props.path);
    };

    render() {
        const {
            // active,
            match,
            location,
            history,
            path,
            label,
            exact,
            strict,
            disabled,
            ...rest
        } = this.props;
        const { hover } = this.state;

        const active = matchPath(location.pathname, {
            path, exact, strict
        });

        const background = active ? "light-1" : hover && !disabled ? "light-1" : null;

        return (
            <Box pad={{vertical: "xsmall", horizontal: "medium"}}
                 style={{cursor: 'pointer'}}
                 background={background}
                 onMouseEnter={() => this.setState({hover: true})}
                 onMouseLeave={() => this.setState({hover: false})}
                 // hoverIndicator="brand"
                 onClick={this.onClick}>
                <Button label={label} plain alignSelf="start" {...rest} />
            </Box>
        );
    }

}

NavMenu.propTypes = {
    location: PropTypes.object.isRequired,
    history: PropTypes.object.isRequired,
    path: PropTypes.string
};

export default withRouter(NavMenu);
