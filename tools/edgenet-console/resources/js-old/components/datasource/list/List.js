import React from "react";
import {Box, InfiniteScroll} from "grommet";

import { ListRow } from './';
import { DataSourceConsumer } from "../../datasource";

const Loading = () =>
    <Box flex="grow" justify="center" align="center">...</Box>;

const List = ({children, onRowClick, show=false}) => {
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
        <DataSourceConsumer>
            {
                ({
                     identifier,
                     items,
                     getItem,
                     currentId,
                     getItems,
                     limit,
                     itemsLoading,
                     selectable,
                     orderable
                }) => {

                    if (itemsLoading) {
                        return <Loading/>;
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
                                step={limit}
                                show={currentIdx}
                                // renderMarker={marker => loading && <Box pad="medium" background="accent-1">{marker}</Box>}
                            >
                                {(item, j) =>
                                    <ListRow key={'items-' + j} item={item} getItem={onRowClick === undefined ? getItem : onRowClick}
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
        </DataSourceConsumer>
    );
};

export default List;
