import React from "react";
import PropTypes from "prop-types";
import { Box } from "grommet";
import { OrderableItem } from "../order";

class ListRow extends React.Component {

    constructor(props) {
        super(props);
        this.state = {
            isMouseOver: false
        }
    }

    handleClick(item) {
        const { getItem } = this.props;

        if (getItem) {
            getItem(item);
        }
    }

    render() {
        let { children } = this.props;
        const { item, isActive, orderable } = this.props;
        const { isMouseOver } = this.state;

        const background = isActive ? 'light-4' : isMouseOver ? 'light-2' : '';

        children = React.cloneElement(children, {
            item: item,
            isActive: isActive,
            isMouseOver: isMouseOver,
        }, null);

        if (orderable) {
            children = <OrderableItem isMouseOver={isMouseOver}
                                      item={item}>{children}</OrderableItem>
        }

        return (
            <Box onMouseEnter={() => this.setState({ isMouseOver: true })}
                 onMouseLeave={() => this.setState({ isMouseOver: false })}
                 onClick={() => this.handleClick(item)}
                 background={background}
                 border={{side:'bottom', color:'light-4'}}
                 flex={false}>
                {children}
            </Box>
        );
    }
}

export default ListRow;
