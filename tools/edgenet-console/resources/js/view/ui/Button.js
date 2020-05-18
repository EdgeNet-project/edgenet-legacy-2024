import React from "react";
import PropTypes from "prop-types";
import { matchPath, withRouter } from "react-router";
import { Box, Button } from "grommet";

class ViewButton extends React.Component {

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

        const background = active ? "white" : hover && !disabled ? "brand" : null;

        return (
            <Box pad={{vertical: "xsmall", horizontal: "medium"}}
                 style={{cursor: 'pointer'}} background={background}
                 onMouseEnter={() => this.setState({hover: true})}
                 onMouseLeave={() => this.setState({hover: false})}
                 // hoverIndicator="brand"
                 onClick={this.onClick}>
                <Button label={label} plain alignSelf="start" {...rest} />
            </Box>
        );
    }

}

ViewButton.propTypes = {
    location: PropTypes.object.isRequired,
    history: PropTypes.object.isRequired,
    path: PropTypes.string
};

export default withRouter(ViewButton);
