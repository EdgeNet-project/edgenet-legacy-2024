import React from "react";
import PropTypes from "prop-types";
import axios from "axios";
import qs from "qs";

import { Searchable } from "./search";
import { Sortable } from "./sort";
import { Filterable } from "./filter";
// import { SelectableContext } from "./selectable";
import { Orderable } from "./order";

const DataSourceContext = React.createContext({});
const DataSourceProvider = DataSourceContext.Provider;
const DataSourceConsumer = DataSourceContext.Consumer;

/**
 * list() -> List items
 * GET /<api>
 *
 * get() -> get item
 * GET /<api>/<id>
 *
 * save() -> save item
 * POST /<source>/<id>
 *
 * delete() -> delete item
 * DELETE /<source>/<id>
 *
 */
class DataSource extends React.Component {

    constructor(props) {
        super(props);

        this.state = {
            source: null,
            id: null,

            items: [],
            total: 0,
            count: 0,
            queryParams: {},
            itemsLoading: true,
            itemsDownloading: false,
            more: true,


            item: null,
            file: null,
            itemLoading: false,
            itemChanged: false,

        };

        this.getItems = this.getItems.bind(this);
        this.downloadItems = this.downloadItems.bind(this);
        this.refreshItems = this.refreshItems.bind(this);

        this.getItem = this.getItem.bind(this);
        this.resetItem = this.resetItem.bind(this);
        this.unsetItem = this.unsetItem.bind(this);
        this.changeItem = this.changeItem.bind(this);
        this.saveItem = this.saveItem.bind(this);
        this.deleteItem = this.deleteItem.bind(this);

        this.attachFile = this.attachFile.bind(this);

        this.setQueryParams = this.setQueryParams.bind(this);
        this.reorderById = this.reorderById.bind(this);
    }

    componentDidMount() {
        const { source, id, sort_by, filter, limit } = this.props;



        // this.setState({
        //     source: source,
        //     id: id
        // });

        id ? this.getItem(id) : this.setQueryParams({
            sort_by: sort_by,
            // filter: filter,
            // limit: limit
        });

    }


    componentDidUpdate(prevProps, prevState, snapshot) {
        // console.log('## ListContext update: ', prevState, '=>', this.state)
        // console.log('## ListContext update: ', prevProps, '=>', this.props)
        //
        const { source, id } = this.props;

        if (prevProps.source !== source) {
            // reloading items
            this.getItems();
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
        if (props.source !== state.source || props.id !== state.id) {
            return {
                source: props.source,
                id: props.id,
                items: [],
                data: [],
                total: 0,
                count: 0,
                queryParams: {},
                itemsLoading: true,
                more: true,

                item: null,
                itemLoading: false,
                itemChanged: false,
                itemSaved: false

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

    getItems(fn) {
        const { source, limit } = this.props;
        const { items, count, queryParams } = this.state;

        if (!source) return false;

        let offset = items.length;

        if (offset < 0) {
            offset = 0;
        }

        if ((offset > 0) && (offset >= count)) {
            this.setState({ more: false });
            return;
        }

        axios.get(source, {
            params: { ...queryParams, offset: offset, limit: limit },
            paramsSerializer: qs.stringify,
        })
            .then(response => {
                const { items } = response.data;
                this.setState({
                    items: items.map(item => { return {...item.spec, name: item.metadata.name}}),
                    total: items.length,
                    count: items.length,
                    itemsLoading: false,
                    // more: (meta.total > 0)
                }, () => (typeof fn === 'function') && fn());
            })
            .catch(error => {

            });
    }

    refreshItems(fn) {
        this.setState({
            items: [],
            total: 0,
            count: 0,
            itemsLoading: true,
        }, () => this.getItems(fn));
    }

    downloadItems(type = '') {
        const { source } = this.props;
        const { queryParams } = this.state;

        this.setState({
            itemsDownloading: true
            }, () => axios.get(
                source + '/' + type, {
                    responseType: 'blob', params: { ...queryParams }, paramsSerializer: qs.stringify
                })
                .then((response) => {
                    const url = window.URL.createObjectURL(new Blob([response.data]));
                    const filename = response.request.getResponseHeader('Content-Disposition').match(/filename="(.+)"/)[1];
                    const link = document.createElement('a');
                    link.href = url;
                    link.setAttribute('download', filename);
                    document.body.appendChild(link);
                    link.click();
                    link.remove();
                    this.setState({ itemsDownloading:false });
                }).catch(error => null)
        );
    }


    getItem(id) {
        const { source } = this.props;
        this.setState({
            itemLoading: true,
            itemChanged: false,
        }, () =>
            axios.get(source + '/' + id)
                .then(({data}) => this.setState({
                    item: DataSource.sanitize(data),
                    itemLoading: false
                }))
        );

    }

    attachFile(file) {
        this.setState({
            file: file
        })
    }

    resetItem() {
        const { item } = this.props;

        if (!item.id) return;

        this.getItem(item.id);

    }

    unsetItem() {
        this.setState({
            item: null,
            itemLoading: false,
            itemChanged: false,
            itemSaved: false
        });
    }

    changeItem(value) {
        // console.log('c',changed.target.name,changed.target.value)
        let changedValue = {};

        if (value.target) {
            changedValue[value.target.name] = value.target.value;
        } else {
            changedValue = value;
        }
        this.setState({
            item: {
                ...this.state.item,
                ...changedValue
            },
            itemChanged: true,
            itemSaved: false
        });
    }

    saveItem(item, fn) {
        const { source } = this.props;
        const { file } = this.state;
        let postData = new FormData();
        for (let key of Object.keys(item)) {
            if (!key) continue;
            postData.append(key, item[key]);
        }

        if (file) {
            postData.append('file', file);
        }

        if (item.id) {
            this.setState({itemChanged: false}, () =>
                axios.post(source + '/' + item.id, postData)
                    .then(({data}) => this.setState({
                        item: DataSource.sanitize(data),
                        itemSaved: true
                    }, () => this.refreshItems(fn)))
                    .catch(() => this.setState({itemChanged: true, itemSaved: false}))
            )
        } else {
            this.setState({itemChanged: false}, () =>
                axios.post(source, postData)
                    .then(({data}) => this.setState({
                        item: DataSource.sanitize(data),
                        itemSaved: true
                    }, () => this.refreshItems(fn)))
                    .catch(() => this.setState({itemChanged: true, itemSaved: false})
                    )
            )
        }
    }

    deleteItem(id) {
        const { source } = this.props;

        this.setState({itemChanged: false, itemLoading: true}, () =>
            axios.delete(source + '/' + id)
                .then(() => {
                    this.setState({
                        itemLoading: false
                    }, this.refreshItems)
                })
                .catch(() => this.setState({itemChanged: true, itemLoading: false})
                )
        )
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

    reorderById(ids) {
        // reorders items by the given array of ids
        const { items } = this.state;
        const { identifier } = this.props;

        let ordered = ids.map(id => items.find(item => item[identifier] === id));
        // console.log(ordered.map(o => o.id))
        this.setState({
            items: ordered
        })

    }

    render() {
        let { children } = this.props;
        const { source, identifier, limit, currentId,
            searchable, filterable, sortable, orderable  } = this.props;
        const {
            items, total, count, itemsLoading, itemsDownloading, more, queryParams,
            item, itemLoading, itemChanged, itemSaved,
            error, errorInfo } = this.state;

        if (error) {
            return error + ' ' + errorInfo;
        }

        if (searchable) {
            const { search } = this.props;
            children = <Searchable source={source} search={search} setQueryParams={this.setQueryParams}>{children}</Searchable>;
        }

        if (sortable) {
            const { sort_by } = this.props;
            children = <Sortable source={source} sort_by={sort_by} setQueryParams={this.setQueryParams}>{children}</Sortable>;
        }

        if (filterable) {
            const { filter } = this.props;
            children = <Filterable source={source} filter={filter} setQueryParams={this.setQueryParams}>{children}</Filterable>;
        }
        //
        // if (selectable) {
        //     children = <SelectableContext identifier={identifier}>{children}</SelectableContext>;
        // }
        //
        if (orderable) {
            children = <Orderable source={source} identifier={identifier} items={items}
                                  handleReorder={(items) => this.setState({items: items})}>{children}</Orderable>
        }

        return (
            <DataSourceProvider value={{
                identifier: identifier,

                source: source,

                items: items,
                total: total,
                count: count,
                itemsLoading: itemsLoading,
                itemsDownloading: itemsDownloading,
                more: more,

                item: item,
                itemLoading: itemLoading,
                itemChanged: itemChanged,
                itemSaved: itemSaved,

                limit: limit,

                currentId: currentId,

                setQueryParams: this.setQueryParams,
                queryParams: queryParams,

                getItems: this.getItems,
                downloadItems: this.downloadItems,
                refreshItems: this.refreshItems,

                getItem: this.getItem,
                resetItem: this.resetItem,
                unsetItem: this.unsetItem,
                changeItem: this.changeItem,
                saveItem: this.saveItem,
                deleteItem: this.deleteItem,

                attachFile: this.attachFile,


                // appendItem: (item) => this.setState({ items: items.concat([item]) }),
                // removeItem: (id) => this.setState({ items: items.filter(item => item[identifier] !== id) }),
                // loading: loading,

                // selectable: selectable,
                // sortable: sortable,
                // searchable: searchable,
                orderable: orderable,
            }}>
                {children}
            </DataSourceProvider>
        );

    }
}

DataSource.propTypes = {
    source: PropTypes.string.isRequired,
    id: PropTypes.any,

    identifier: PropTypes.string,
    sort_by: PropTypes.arrayOf(
        PropTypes.shape({
            name: PropTypes.string,
            direction: PropTypes.oneOf(['asc','desc'])
        })
    ),
    filter: PropTypes.object,
    limit: PropTypes.number,

    sortable: PropTypes.bool,
    filterable: PropTypes.bool,
    selectable: PropTypes.bool,
    searchable: PropTypes.bool,


};

DataSource.defaultProps = {
    identifier: 'name',
    sort_by: [],
    filter: {},
    limit: 30,
    sortable: false,
    filterable: false,
    selectable: false,
    searchable: false,
};

export { DataSource, DataSourceContext, DataSourceConsumer };
