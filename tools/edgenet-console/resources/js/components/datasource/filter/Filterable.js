import React from "react";
import PropTypes from "prop-types";
import { hash, getSession, setSession } from "../util";

const FilterableContext = React.createContext({
    filter: {}
});
const FilterableProvider = FilterableContext.Provider;
const FilterableConsumer = FilterableContext.Consumer;

class Filterable extends React.Component {

    constructor(props) {
        super(props);
        this.state = {
            filter: (props.filter !== undefined) ? props.filter : {}
        };

        this.hash = hash(props.source);

        this.apply = this.apply.bind(this);
        this.setFilter = this.setFilter.bind(this);
        this.addFilter = this.addFilter.bind(this);
        this.removeFilter = this.removeFilter.bind(this);
        this.clearFilter = this.clearFilter.bind(this);
        this.hasFilter = this.hasFilter.bind(this);
        this.countFilter = this.countFilter.bind(this);

    }

    componentDidMount() {
        const filter = getSession(this.hash, 'filter');
        if (filter) {
            this.apply({
                filter: filter
            });
        }
    }

    componentDidCatch(error, errorInfo) {
        this.setState({
            error: error,
            errorInfo: errorInfo
        })
    }

    apply(state) {
        this.setState(state, () => {
            this.props.setQueryParams(state);
            setSession(this.hash , 'filter', state.filter)
        });
    }

    /**
     * Updates filter name with the content of value
     * value must be an array of values or a string
     * @param name
     * @param value
     */
    setFilter(name, value) {
        const { filter } = this.state;

        this.apply({
            filter: {
                ...filter,
                [name]: value
            }
        });
    }

    /**
     * Add a filter as filter[name][]=value
     */
    addFilter(name, value) {
        const { filter } = this.state;

        // console.log(name,value)
        // let f = [];
        // if (filter[name]) {
        //     f = filter[name];
        // }
        if (filter[name] === undefined) {
            filter[name] = [];
        }

        if (filter[name].find(f => f === value)) {
            return;
        }

        this.setFilter(name, filter[name].concat(value));
    }

    /**
     * removes a filter value or, if value is
     * null removes the filter completely
     */
    removeFilter(name, value=null) {
        let { filter } = this.state;

        if (value !== null) {
            if (filter[name] === undefined) {
                return;
            }
            filter[name] = filter[name].filter(f => f !== value);
        } else {
            delete(filter[name]);
        }

        this.apply({ filter: { ...filter }});
    }

    clearFilter() {
        this.apply({ filter: {} });
    }

    hasFilter(name=null, value=null) {
        let { filter } = this.state;

        if (name === null) {
            return Object.keys(filter).length > 0 && filter.constructor === Object;
        }

        return filter[name] !== undefined && (value === null || filter[name].includes(value));
    }

    countFilter(names=[]) {
        let { filter } = this.state;

        if (names.length > 0) {
            let count = 0;
            for (const name of Object.keys(filter)) {
                count++;
            }
            return count;
        } else {
            return Object.keys(filter).length;
        }
    }

    render() {
        if (this.state.error) {
            return this.state.error + ' ' + this.state.errorInfo;
        }

        return (
            <FilterableProvider value={{
                filter: this.state.filter,
                addFilter: this.addFilter,
                removeFilter: this.removeFilter,
                setFilter: this.setFilter,
                clearFilter: this.clearFilter,
                hasFilter: this.hasFilter,
                countFilter: this.countFilter,
            }}>
                {this.props.children}
            </FilterableProvider>
        );

    }
}

Filterable.propTypes = {
    setQueryParams: PropTypes.func.isRequired,
    filter: PropTypes.object
};

Filterable.defaultProps = {
};

export { Filterable, FilterableContext, FilterableConsumer };
