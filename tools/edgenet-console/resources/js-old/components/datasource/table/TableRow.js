import React from "react";
import PropTypes from "prop-types";
import { TableRow as TableRowGrommet, TableCell } from "grommet";
// import { SelectableItem } from "./selectable";
// import { OrderableItem } from "./orderable";

class TableRow extends React.Component {

    constructor(props) {
        super(props);
        this.state = {
            isMouseOver: false
        }

        this.handleRowClick = this.handleRowClick.bind(this);
    }

    handleRowClick() {
        const { item, onClick } = this.props;

        if (onClick) {
            onClick(item.id);
        }
    }

    render() {
        let { children } = this.props;
        const { item, isActive, selectable, orderable } = this.props;
        const { isMouseOver } = this.state;
        const background = isActive ? 'light-4' : isMouseOver ? 'light-2' : '';

        // children = React.cloneElement(children, {
        //     item: item,
        //     isActive: isActive,
        //     isMouseOver: isMouseOver,
        // }, null);

        // if (selectable) {
        //     children = <SelectableItem isMouseOver={isMouseOver}
        //                                isActive={isActive}
        //                                item={item}>{children}</SelectableItem>
        // }
        //
        // if (orderable) {
        //     children = <OrderableItem isMouseOver={isMouseOver}
        //                               item={item}>{children}</OrderableItem>
        // }

        return (
            <TableRowGrommet
                onMouseEnter={() => this.setState({ isMouseOver: true })}
                onMouseLeave={() => this.setState({ isMouseOver: false })}
                onClick={this.handleRowClick}

                border={{side:'bottom', color:'light-4'}}
                flex={false}>
                {React.Children.map(children, (child, i) =>
                    <TableCell key={child.key + i} background={background} >
                        { React.cloneElement(child, {
                            item,
                            isActive: isActive,
                            isMouseOver: isMouseOver,
                        }) }
                    </TableCell>
                )}
            </TableRowGrommet>
        );
    }
}

export default TableRow;
