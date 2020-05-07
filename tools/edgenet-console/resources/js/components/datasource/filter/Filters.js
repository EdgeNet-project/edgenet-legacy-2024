import React from "react";
import PropTypes from "prop-types";

import { Box, Button } from "grommet";
import { ClearOption, Filter } from "grommet-icons";

import { FilterableConsumer } from "./Filterable";
import { ToolbarTab } from "../toolbar";

const Filters = ({children, label="Effacer les filtres"}) =>
    <FilterableConsumer>
        {
            ({filter, clearFilter, hasFilter}) => filter &&
                <Box flex={false} pad="small" gap="small">
                    {children}
                    <Box align="start" border={{side:"top",color:"dark-6"}} pad={{top:"small"}}>
                        <Button plain hoverIndicator disabled={!hasFilter()}
                                label={label}
                                icon={<ClearOption />}
                                onClick={clearFilter} />
                    </Box>
                </Box>
        }
    </FilterableConsumer>;

Filters.Tab = ({label="Filters", ...props}) =>
    <FilterableConsumer>
        {({hasFilter, countFilter}) => hasFilter &&
            <ToolbarTab icon={<Filter />} label={label} count={countFilter()} {...props} />
        }
    </FilterableConsumer>;

export default Filters;
