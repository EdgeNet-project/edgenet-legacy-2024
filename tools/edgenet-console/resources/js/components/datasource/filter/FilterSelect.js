import React from 'react';
import { Select } from "../components";

import { FilterableConsumer } from "./Filterable";

const FilterSelect = ({name, label, source, limit, multiple, labelKey, valueKey}) =>
    <FilterableConsumer>
        {
            ({filter, addFilter, setFilter, removeFilter, clearFilter}) =>
                <Select name={name}
                        label={label}
                        value={filter[name]}
                        source={source}
                        limit={limit}
                        onSelect={(value) => multiple ? addFilter(name, value) : setFilter(name, value)}
                        onRemove={(value) => multiple ? removeFilter(name, value) : removeFilter(name)}
                        multiple={multiple}
                        labelKey={labelKey}
                        valueKey={valueKey}
                />
        }
    </FilterableConsumer>;

export default FilterSelect;