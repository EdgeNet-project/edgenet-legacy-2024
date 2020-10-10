import React, { Component } from 'react';

const ReactContext = React.createContext({});
const OrderableProvider = ReactContext.Provider;
const OrderableConsumer = ReactContext.Consumer;

class OrderableContext extends Component {
    constructor(props) {
        super(props);
        this.state = {
            mouseover: false,
            draggable: false,

            elementOver: false,

        };

        this.dragStart = this.dragStart.bind(this);
        this.dragEnd = this.dragEnd.bind(this);
        this.dragOver = this.dragOver.bind(this);

    }

    dragStart(ev) {
        //console.log(ev);

        // this.dragged = Number(ev.currentTarget.dataset.id);
        ev.dataTransfer.effectAllowed = 'move';

        // Firefox requires calling dataTransfer.setData
        // for the drag to properly work
        ev.dataTransfer.setData("text/html", null);
    }

    dragOver(ev, id) {
        ev.preventDefault();

        if (id !== this.state.elementOver) {
            // const items = this.state.list;
            const over = ev.currentTarget
            console.log(id)

            this.setState({elementOver: id})
            // const dragging = this.state.dragging;
            // const from = isFinite(dragging) ? dragging : this.dragged;
            // let to = Number(over.dataset.id);

            // items.splice(to, 0, items.splice(from,1)[0]);
            //console.log(over)
            //this.sort(items, to);
        }
    }
    dragEnd(ev) {
        this.setState({elementOver: false})
        //console.log(ev)
        // this.sort(this.state.list, undefined);
    }

    render() {
        const { children } = this.props;
        const { elementOver } = this.state;


        return (
            <OrderableProvider value={{
                onDragStart: this.dragStart,
                onDragOver: this.dragOver,
                onDragEnd: this.dragEnd,
                elementOver: elementOver
            }}>
                {children}
            </OrderableProvider>
        );


    }
}

export { OrderableContext, OrderableConsumer };
