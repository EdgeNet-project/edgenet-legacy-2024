import React from "react";
import { Box, InfiniteScroll,
    Table as TableGrommet,
    TableBody,
    TableRow as TableRowGrommet,
    TableCell,
    TableHeader } from "grommet";

import TableRow from './TableRow';
import { DataSourceConsumer } from "./.";

const Loading = () =>
    <Box flex="grow" justify="center" align="center">...</Box>;

const Table = ({children, columns, show=false, fill=true, onClick=null}) => {

    return (
        <DataSourceConsumer>
            {
                ({identifier, items, currentId, getItems, limit, itemsLoading, selectable, orderable}) => {

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
                            <TableGrommet>
                                {columns && <TableHeader>
                                    <TableRowGrommet>
                                        {columns.map(column =>
                                            <TableCell key={column} scope="col" border="bottom">
                                                {column}
                                            </TableCell>
                                        )}
                                    </TableRowGrommet>
                                </TableHeader>}
                                <TableBody>
                                    <InfiniteScroll items={items} onMore={getItems} step={limit}
                                                    show={currentIdx}
                                                    renderMarker={marker => (
                                                        <TableRowGrommet>
                                                            <TableCell colSpan={columns.length}>{marker}</TableCell>
                                                        </TableRowGrommet>
                                                    )}
                                    >
                                        {(item, j) =>
                                            <TableRow key={'items-t-'+item[identifier]+'-j'} item={item}
                                                      selectable={selectable} orderable={orderable}
                                                      isActive={item[identifier] === currentId}
                                                      onClick={onClick}>
                                                {children}
                                            </TableRow>
                                        }
                                    </InfiniteScroll>
                                </TableBody>
                            </TableGrommet>
                        </Box>
                    )
                }
            }
        </DataSourceConsumer>
    );
};

export default Table;
