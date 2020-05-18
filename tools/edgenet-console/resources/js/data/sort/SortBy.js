import React from "react";
import PropTypes from "prop-types";

import { Box } from "grommet";
import { Ascend, Descend } from "grommet-icons";
import { ToolbarTab, ToolbarButton } from "../toolbar";
import { SortableConsumer } from "./Sortable";

const SortBy = ({options}) =>
        <SortableConsumer>
            {({toggleSortBy, isSortBy, isSortByAsc, resetSortBy}) => toggleSortBy !== undefined &&
                <Box flex={false} direction="row" gap="small" pad="small" wrap>
                    {
                        options.map(({name, label}) => {
                                let color = isSortBy(name) ? "brand" : null;
                                return <ToolbarButton key={name} label={label}
                                               icon={isSortBy(name) ? isSortByAsc(name) ?
                                                   <Ascend color={color}/> : <Descend color={color}/> : null}
                                               onClick={() => toggleSortBy(name)}
                                />;
                            }
                        )
                    }
                </Box>
            }
        </SortableConsumer>;

SortBy.Tab = ({options, label, ...props}) =>
    <SortableConsumer>
        {
            ({sort_by}) => {
                if (sort_by === undefined) return null;
                let { label } = sort_by.length > 0 ? options.find(({name}) => name === sort_by[0].name) : { label: label };
                let direction = sort_by.length > 0 ? sort_by[0].direction : 'asc';
                return <ToolbarTab icon={(direction === 'asc' ? <Ascend /> : <Descend />)}
                                   label={label} {...props} />;
            }
        }
    </SortableConsumer>;
//
//
// SortableToolbar.propTypes = {
//     options: PropTypes.arrayOf(
//         PropTypes.exact({
//             label: PropTypes.string,
//             name: PropTypes.string,
//         }),
//     ),
// };
//
// SortableToolbar.defaultProps = {
//
// };

export default SortBy;
