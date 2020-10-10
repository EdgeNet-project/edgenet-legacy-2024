import React from "react";
import PropTypes from "prop-types";
import { Box } from "grommet";
import { OrderableConsumer } from "./Orderable";
import {Drag} from "grommet-icons/es6";

class OrderableItem extends React.Component {

    constructor(props) {
        super(props);
        this.state = {
            cursor: 'auto',
            draggable: false
        };

        this.setCursorGrab = this.setCursorGrab.bind(this);
        this.setCursorGrabbing = this.setCursorGrabbing.bind(this);
        this.setCursorAuto = this.setCursorAuto.bind(this);

    }

    setCursorGrab() {
        this.setState({cursor: 'grab', draggable: false})
    }

    setCursorGrabbing() {
        this.setState({cursor: 'grabbing', draggable: true})
    }

    setCursorAuto() {
        this.setState({cursor: 'auto'})
    }

    render() {
        let { children, item, isMouseOver } = this.props;
        const { cursor, draggable } = this.state;

        return (
            <OrderableConsumer>
                { ({onDragStart, onDragOver, onDragEnd, elementOver}) =>
                    <Box draggable={draggable}
                         onDragStart={onDragStart}
                         onDragOver={(ev) => onDragOver(ev, item.id)}
                         onDragEnd={onDragEnd}
                         border={{side:'top', size:'small', color: elementOver === item.id ? 'brand' : 'inherit'}}
                         direction="row" background={draggable ? 'white' : ''}>
                        <Box justify="center" align="center" style={{cursor:cursor}} pad={{right:"xsmall"}}>
                            <Drag onMouseOver={this.setCursorGrab} onMouseOut={this.setCursorAuto}
                                  onMouseDown={this.setCursorGrabbing} onMouseUp={this.setCursorGrab} />

                        </Box>
                        {children}
                    </Box>
                }
            </OrderableConsumer>
        );
    }
}

OrderableItem.propTypes = {
    // item: PropTypes.object.isRequired
};

OrderableItem.defaultProps = {
};

export default OrderableItem;
