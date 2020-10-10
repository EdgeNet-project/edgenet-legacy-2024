import React from "react";
import PropTypes from "prop-types";
import { hash, getSession, setSession } from "../util";

const SortableContext = React.createContext({
    sort_by: []
});
const SortableProvider = SortableContext.Provider;
const SortableConsumer = SortableContext.Consumer;

class Sortable extends React.PureComponent {

    constructor(props) {
        super(props);
        this.state = {
            sort_by: (props.sort_by !== undefined) ? props.sort_by : []
        };

        this.hash = hash(props.source);

        this.apply = this.apply.bind(this);
        this.setSortBy = this.setSortBy.bind(this);
        this.delSortBy = this.delSortBy.bind(this);
        this.toggleSortBy = this.toggleSortBy.bind(this);
        this.clearSortBy = this.clearSortBy.bind(this);
        this.resetSortBy = this.resetSortBy.bind(this);
        this.isSortBy = this.isSortBy.bind(this);
        this.isSortByAsc = this.isSortByAsc.bind(this);
    }

    componentDidMount() {
        const sort_by = getSession(this.hash, 'sort_by');
        if (sort_by) {
            this.apply({
                sort_by: sort_by
            });
        }
    }

    componentDidUpdate(prevProps, prevState, snapshot) {
        // console.log('## SortableContext update: ', prevState.sort_by, '=>', this.state.sort_by)
    }

    apply(state) {
        this.setState(state,
            () => {
                this.props.setQueryParams(state);
                setSession(this.hash , 'sort_by', state.sort_by)
            }
        );
    }

    getSortBy(name) {
        return this.state.sort_by.find(s => s.name === name);
    }

    setSortBy(name, direction = 'asc') {
        this.apply({
            sort_by: [ { name: name, direction: direction } ]
        });
    }

    delSortBy(name) {
        this.apply({
            sort_by: this.state.sort_by.filter(s => s.name !== name)
        });
    }

    toggleSortBy(name) {
        let sort_by = this.getSortBy(name);

        (sort_by === undefined) ?
            this.setSortBy(name) :
            this.setSortBy(name, sort_by.direction === 'asc' ? 'desc' : 'asc');
    }

    resetSortBy() {
        // sets the defaults
        let { sort_by } = this.props;

        this.apply({
            sort_by: (sort_by !== undefined) ? sort_by : []
        });
    }

    clearSortBy() {
        this.apply({ sort_by: [] });
    }

    isSortBy(name) {
        return this.state.sort_by.some(s => s.name === name);
    }

    isSortByAsc(name) {
        let sort_by = this.getSortBy(name);

        return (sort_by !== undefined) ? sort_by.direction === 'asc' : undefined;
    }

    render() {
        return (
            <SortableProvider value={{
                sort_by: this.state.sort_by,
                setSortBy: this.setSortBy,
                delSortBy: this.delSortBy,
                toggleSortBy: this.toggleSortBy,
                clearSortBy: this.clearSortBy,
                resetSortBy: this.resetSortBy,
                isSortByAsc: this.isSortByAsc,
                isSortBy: this.isSortBy,
            }}>
                {this.props.children}
            </SortableProvider>
        );
    }
}


Sortable.propTypes = {
    setQueryParams: PropTypes.func.isRequired,
    sort_by: PropTypes.arrayOf(
        PropTypes.shape({
            name: PropTypes.string,
            direction: PropTypes.oneOf(['asc','desc'])
        })
    )
};

Sortable.defaultProps = {
};

export { Sortable, SortableContext, SortableConsumer };
