import React from "react";
import PropTypes from "prop-types";
import { hash, getSession, setSession, clearSession } from "../util";

const SearchableContext = React.createContext({
    search: ''
});
const SearchableProvider = SearchableContext.Provider;
const SearchableConsumer = SearchableContext.Consumer;

class Searchable extends React.PureComponent {

    constructor(props) {
        super(props);
        this.state = {
            search: (props.search !== undefined) ? props.search : ''
        };

        this.hash = hash(props.source);

        this.searchDelay = null;

        this.setSearch = this.setSearch.bind(this);
        this.clearSearch = this.clearSearch.bind(this);
    }

    componentDidMount() {
        this.setSearch(getSession(this.hash, 'search'));
    }

    setSearch(string) {

        if (string == null) return;

        this.setState({ search: string });

        if (string.length > 1 && string.length !== 0) {
            clearTimeout(this.searchDelay);

            this.searchDelay = setTimeout(() => {
                this.props.setQueryParams({search: string});
                setSession(this.hash, 'search', string)
            }, 400);
        }

    }

    clearSearch() {
        this.setState({ search: '' },
            () => {
                this.props.setQueryParams({search: ''});
                clearSession(this.hash, 'search');
            });
    }

    render() {
        return (
            <SearchableProvider value={{
                search: this.state.search,
                setSearch: this.setSearch,
                clearSearch: this.clearSearch
            }}>
                {this.props.children}
            </SearchableProvider>
        );
    }

}

Searchable.propTypes = {
    setQueryParams: PropTypes.func.isRequired,
    search: PropTypes.string
};

Searchable.defaultProps = {
};

export { Searchable, SearchableContext, SearchableConsumer }