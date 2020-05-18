import React from "react";
import { Box, InfiniteScroll } from "grommet";

import { DataConsumer } from "../.";
import { OrderableItem } from "../order";

const Loading = () =>
    <Box flex="grow" justify="center" align="center">...</Box>;

class ListRow extends React.Component {

    constructor(props) {
        super(props);
        this.state = {
            isMouseOver: false
        }
    }

    render() {
        let { children } = this.props;
        const { item, isActive, orderable, onClick } = this.props;
        const { isMouseOver } = this.state;

        const background = isActive ? 'light-4' : isMouseOver ? 'light-2' : 'light-1';

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
                 onClick={() => onClick(item)}
                 background={background}
                 border={{side:'bottom', color:'light-4'}}
                 flex={false}>
                {children}
            </Box>
        );
    }
}

const List = ({children, onClick, show=false}) => {
    //
    // let ListComponent = null;
    // if (component) {
    //     ListComponent = component;
    // } else if (children) {
    //     ListComponent = React.cloneElement(children, null);
    // } else {
    //     return null;
    // }

    return (
        <DataConsumer>
            {
                ({
                     identifier,
                     items,
                     per_page,
                     itemsLoading,

                     getItem,
                     currentId,
                     getItems,

                     selectable,
                     orderable
                }) => {

                    if (itemsLoading) {
                        return <Loading />;
                    }

                    if (items.length === 0) {
                        return <Box pad="small" align="center">No items found</Box>;
                    }

                    if (currentId) {
                        if (!isNaN(currentId)) {
                            // check if it is a number
                            currentId = parseInt(currentId);
                        }
                    }
                    let currentIdx = null;
                    if (show && currentId) {
                        currentIdx = items.findIndex(item => item[identifier] === currentId)
                    }

                    return (
                        <Box overflow="auto">
                            <InfiniteScroll
                                items={items}
                                onMore={getItems}
                                step={per_page}
                                // show={currentIdx}
                                // renderMarker={marker => itemsLoading && <Box pad="medium" background="accent-1">{marker}</Box>}
                            >
                                {(item, j) =>
                                    <ListRow key={'items-' + j} item={item} onClick={onClick === undefined ? getItem : onClick}
                                             selectable={selectable} orderable={orderable}
                                             isActive={item[identifier] === currentId}>
                                        {children}
                                    </ListRow>
                                }
                            </InfiniteScroll>
                        </Box>
                    )
                }
            }
        </DataConsumer>
    );
};

export default List;
