import React, { Component }  from "react";
import PropTypes from "prop-types";
import { Box } from "grommet";
import { Drag, Pan } from "grommet-icons";
import { OrderableConsumer } from "./Orderable";

class OrderableItemIcon extends Component {

    constructor(props) {
        super(props);
        this.state = {
            grabbing: false
        };
    }

    render() {
        const { onMouseDown } = this.props;
        const { grabbing } = this.state;

        return (
            <Box justify="center" align="center" pad={{left:"xsmall"}}
                 style={{cursor: grabbing ? 'grabbing' : 'grab'}}
                 onMouseDown={() => this.setState({grabbing: true}, onMouseDown)}
                 onMouseUp={() => this.setState({grabbing: false})}>
                <Drag />
            </Box>
        );
    }
}

class OrderableItem extends Component {

    constructor(props) {
        super(props);

        this.ref = React.createRef();
    }

    render() {
        let { children, item } = this.props;

        return (
            <OrderableConsumer>
                { ({handleMouseDown, isDragged, draggedStyle, gapStyle}) => {
                    const moving = isDragged(item);
                    return (
                        <React.Fragment>
                            {moving && <Box style={gapStyle} />}
                            <Box ref={this.ref} direction="row"
                                 style={moving ? draggedStyle : {userSelect:'none'}}>
                                <OrderableItemIcon onMouseDown={() => handleMouseDown(this.ref.current.getBoundingClientRect(), item)} />
                                {children}
                            </Box>
                        </React.Fragment>
                    );


                } }
            </OrderableConsumer>
        );
    }
}

OrderableItem.propTypes = {
    item: PropTypes.object.isRequired
};

OrderableItem.defaultProps = {
};

export default OrderableItem;
