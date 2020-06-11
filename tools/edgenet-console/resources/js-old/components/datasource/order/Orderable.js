import React, { Component } from 'react';
import axios from "axios";

const OrderableContext = React.createContext({});
const OrderableProvider = OrderableContext.Provider;
const OrderableConsumer = OrderableContext.Consumer;

class Orderable extends Component {
    constructor(props) {
        super(props);
        this.state = {
            // the item dragged
            dragged: false,

            draggedIndex: false,
            updatedIndex: false,

            previousOrder: [],

            origin: {},

            gapStyle: {},
            draggedStyle: {}
        };

        this.handleMouseMove = this.handleMouseMove.bind(this);
        this.handleMouseDown = this.handleMouseDown.bind(this);
        this.handleMouseUp = this.handleMouseUp.bind(this);
        this.isDragged = this.isDragged.bind(this);
    }

    componentDidUpdate(prevProps, prevState, snapshot) {

        if (prevState.dragged !== this.state.dragged) {
            if (this.state.dragged !== false) {
                window.addEventListener('mousemove', this.handleMouseMove);
                window.addEventListener('mouseup', this.handleMouseUp);
            } else {
                window.removeEventListener('mousemove', this.handleMouseMove);
                window.removeEventListener('mouseup', this.handleMouseUp);
            }
        }

        if (prevState.updatedIndex !== this.state.updatedIndex && this.state.updatedIndex !== false) {
            /**
             * reorders the items in the list
             */
            const { items, identifier, handleReorder } = this.props;
            // swap items
            const reorderedItems = items.filter(item => item[identifier] !== this.state.dragged[identifier]);
            reorderedItems.splice(this.state.updatedIndex, 0, this.state.dragged);

            if (handleReorder) {
                handleReorder(reorderedItems);
            }
        }
    }

    handleMouseMove({clientX, clientY}) {
        /**
         * mouse moving (dragging) updates item position on screen
         * and the index of the item it is over on the list
         */
        const { origin, draggedStyle } = this.state;
        const updatedIndex = Math.round((clientY - origin.y) / origin.height) + this.state.draggedIndex;

        // console.log(draggedIndex, updatedIndex)

        if (updatedIndex < 0 || updatedIndex > this.props.items.length - 1) {
            // doesn't drag outside of the list
            return;
        }

        this.setState({
            updatedIndex: updatedIndex,
            draggedStyle: {
                ...draggedStyle,
                top: clientY - origin.height / 2,
                left: origin.x,
            }
        });

    }

    handleMouseDown({x, y, height, width, top, bottom, left, right}, item) {
        const { items, identifier } = this.props;
        const currentOrder = items.map(item => item[identifier]);

        this.setState({
            dragged: item,
            draggedIndex: currentOrder.indexOf(item[identifier]),
            previousOrder: currentOrder,
            origin: {
                x: x,
                y: y,
                height: height,
                width: width
            },
            gapStyle: {
                height: height,
                minHeight: height,
                backgroundColor: 'white',
                border: '2px dashed #CECECE'
            },
            draggedStyle: {
                zIndex: 100,
                cursor: 'grabbing',
                backgroundColor: 'white',
                position: 'absolute',
                width: width,
                height: height,
                boxShadow: '0 5px 10px rgba(0, 0, 0, 0.15)',
                userSelect:'none'
            }
        });
    }

    handleMouseUp() {
        const { previousOrder } = this.state;
        const { items, identifier, handleReorder, source } = this.props;

        const currentOrder = items.map(item => item[identifier]);

        if ((previousOrder.length !== currentOrder.length) || !currentOrder.every((id, index) => id === previousOrder[index])) {
            axios.patch('/' + source + '/reorder', {
            order: currentOrder
            }).catch(() => handleReorder(previousOrder.map(id => items.find(item => item[identifier] === id))))
        }
        // compare previous id list
        this.setState({
            dragged: false,
            draggedIndex: false,
            updatedIndex: false,
            origin: {}
        });

    }

    isDragged(item) {
        return item[this.props.identifier] === this.state.dragged[this.props.identifier];
    }

    render() {
        return (
            <OrderableProvider value={{
                draggedStyle: this.state.draggedStyle,
                gapStyle: this.state.gapStyle,
                isDragged: this.isDragged,
                handleMouseDown: this.handleMouseDown,
            }}>
                {this.props.children}
            </OrderableProvider>
        );

    }
}

export { Orderable, OrderableContext, OrderableConsumer };
