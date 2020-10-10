import React from "react";
import PropTypes from "prop-types";
import { withRouter } from "react-router";

import { Anchor } from "grommet";

class NavigationAnchor extends React.Component {

    constructor(props) {
        super(props);
        this.onClick = this.onClick.bind(this);
    }

    onClick(event) {
        const { history, path } = this.props;
        event.preventDefault();
        history.push(path);
        if (this.props.onClick) {
            this.props.onClick();
        }
    };

    render() {
        const {
            exact, match, location, history, path, strict,
            label, children, onClick, ...rest
        } = this.props;

        return (
                <Anchor onClick={this.onClick} label={label} {...rest}>{children}</Anchor>
        );
    }

}

NavigationAnchor.propTypes = {
    location: PropTypes.object.isRequired,
    history: PropTypes.object.isRequired,
    path: PropTypes.string.isRequired
};

export default withRouter(NavigationAnchor);