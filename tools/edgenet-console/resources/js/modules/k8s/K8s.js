import React from "react";
import PropTypes from "prop-types";
import axios from "axios";
import qs from "qs";

const K8sContext = React.createContext({});
const K8sConsumer = K8sContext.Consumer;

class K8s extends React.Component {

    constructor(props) {
        super(props);

        this.state = {
            resource: null,
            name: null,

            items: [],
            metadata: [],


            queryParams: {},
            loading: true,

        };

        this.get = this.get.bind(this);

        this.push = this.push.bind(this);
        this.pull = this.pull.bind(this);

    }

    componentDidMount() {
        // const { url, id, sort_by, filter, limit } = this.props;



        // this.setState({
        //     source: source,
        //     id: id
        // });

        // id ? this.getItem(id) : this.setQueryParams({
        //     sort_by: sort_by,
        //     // filter: filter,
        //     // limit: limit
        // });

        this.get()
    }


    componentDidUpdate(prevProps, prevState, snapshot) {
        // console.log('## ListContext update: ', prevState, '=>', this.state)
        // console.log('## ListContext update: ', prevProps, '=>', this.props)
        //
        const { resource, id } = this.props;

        if (prevProps.resource !== resource) {
            // reloading items
            this.get();
        }

        if (prevProps.id !== id) {
            // reloading item
            this.get(id);
        }
    }

    componentDidCatch(error, errorInfo) {
        this.setState({
            error: error,
            errorInfo: errorInfo
        })
    }

    static getDerivedStateFromProps(props, state) {
        if (props.resource !== state.resource || props.id !== state.id) {
            return {
                resource: props.resource,
                id: props.id,

                items: [],
                current_page: 0,
                last_page: 1,
                per_page: null,
                total: 0,

                queryParams: {},
                loading: true,

            };
        }

        return null;
    }

    componentWillUnmount() {
    }

    /**
     * Sets the query params for the list request.
     */
    setQueryParams(params) {
        let queryParams = {};
        if (params.sort_by !== undefined) {
            queryParams = {
                sort_by: Object.fromEntries(params.sort_by.map(s => [s.name, s.direction]))
            };
        }
        if (params.filter !== undefined) {
            queryParams = {
                ...queryParams, filter: params.filter
            };
        }

        // if (params.limit !== undefined) {
        //     queryParams = {
        //         ...queryParams, limit: params.limit
        //     };
        // }

        if (params.search !== undefined) {
            queryParams = {
                ...queryParams, search: params.search
            };
        }

        this.setState({
            items: [],
            queryParams: { ...this.state.queryParams, ...queryParams }
        }, this.refreshItems);
    }

    get() {
        const { api } = this.props;
        const { items, current_page, last_page, queryParams } = this.state;

        if (!api) return false;

        // if (current_page >= last_page) return;

        axios.get(api.server + api.url, {
            params: { ...queryParams, page: current_page + 1 },
            paramsSerializer: qs.stringify,
        })
            .then(({data}) => {
                this.setState({
                    ...data, loading: false
                });
            })
            .catch(error => {
                console.log(error)
            });
    }


    push(item) {
        const { items } = this.state;
        this.setState({
            items: items.concat([item])
        });
    }

    pull(item) {
        const { items } = this.state;
        this.setState({
            items: items.filter(i => i.id !== item.id)
        });
    }

    refresh() {
        this.setState({
            items: [],
            metadata: [],
            loading: true,
        }, this.get);
    }

    static sanitize(data) {
        /**
         * see:
         * https://github.com/facebook/react/issues/11417
         * https://github.com/reactjs/rfcs/pull/53
         *
         */

        Object.keys(data).forEach((key, idx) => {
            if (data[key] === null) {
                data[key] = '';
            }
        });

        return data;

    }


    render() {
        let { children } = this.props;
        const { resource } = this.props;
        const {
            items, metadata,

            loading,
            error, errorInfo } = this.state;

        if (error) {
            return error + ' ' + errorInfo;
        }

        return (
            <K8sContext.Provider value={{

                resource: resource,

                items: items,


                loading: loading,


                get: this.get,
                push: this.pushItem,
                pull: this.pullItem,

            }}>
                {children}
            </K8sContext.Provider>
        );

    }
}

K8s.propTypes = {
    api: PropTypes.object.isRequired,
    id: PropTypes.any,
};

K8s.defaultProps = {

};

export { K8s, K8sContext, K8sConsumer };
