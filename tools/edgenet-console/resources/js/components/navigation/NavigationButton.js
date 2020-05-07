import React from "react";
import PropTypes from "prop-types";
import { matchPath, withRouter } from "react-router";
import { Button } from "grommet";

class NavigationButton extends React.Component {

    constructor(props) {
        super(props);
        this.onClick = this.onClick.bind(this);
    }

    onClick(event) {
        const { history, path } = this.props;
        event.preventDefault();
        history.push(path);
    };

    render() {
        const { match, location, history, path, exact, strict, label, disabled, ...rest } = this.props;

        const active = matchPath(location.pathname, {
            path, exact, strict
        });

        return (
                <Button label={label} color={active && active.isExact ? 'light-1' : 'dark-1'}
                        disabled={disabled} onClick={!disabled ? this.onClick : undefined}
                        {...rest}
                />
        );
    }

}

NavigationButton.propTypes = {
    location: PropTypes.object.isRequired,
    history: PropTypes.object.isRequired,
    path: PropTypes.string
};

export default withRouter(NavigationButton);
