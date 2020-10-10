import React from 'react';
import propTypes from "prop-types";
import axios from "axios";
import { Box, Text, Select as GrommetSelect } from "grommet";
import { FormClose } from "grommet-icons";

const MultipleSelectedValue = ({ value, labelKey, onRemove }) =>
    <Box onClick={event => {
        event.preventDefault();
        event.stopPropagation();
        onRemove(value);
    }} onFocus={event => event.stopPropagation()} plain>
        <Box align="center" direction="row" gap="xsmall"
             pad={{vertical: "xsmall", horizontal: "small"}}
             margin="xsmall" background="neutral-3" round="large">
            <Text size="small" color="white">{value[labelKey]}</Text>
            <Box background="white" round="full" margin={{left: "xsmall"}}>
                <FormClose color="brand" size="small" />
            </Box>
        </Box>
    </Box>;


const SelectedValue = ({value, labelKey, onRemove}) =>
    <Box pad={{vertical: "xsmall", horizontal: "xsmall"}} margin="xsmall" direction="row">
        <Box onClick={event => {
            event.preventDefault();
            event.stopPropagation();
            onRemove(value);
        }} onFocus={event => event.stopPropagation()}>
            <FormClose />
        </Box>
        <Text>{value[labelKey]}</Text>
    </Box>;

const Placeholder = ({placeholder}) =>
    <Box pad={{vertical: "xsmall", horizontal: "small"}} margin="xsmall">
        {placeholder}&nbsp;
    </Box>;

class Select extends React.Component {

    constructor(props) {
        super(props);
        this.state = {
            options: [],
            count: 0,
            total: 0,
            // holds the filtered options
            // when searching
            searchText: '',
            search: [],
            // mixed value (array or value)
            selected: '',
        };

        this.selectValue = this.selectValue.bind(this);
        this.removeValue = this.removeValue.bind(this);
        this.loadOptions = this.loadOptions.bind(this);
        this.setSelected = this.setSelected.bind(this);

        this.search = this.search.bind(this);
        this.localSearch = this.localSearch.bind(this);
        this.remoteSearch = this.remoteSearch.bind(this);
    }

    componentDidMount() {
        this.loadOptions()
    }

    loadOptions() {
        const { source, limit, labelKey, valueKey } = this.props;
        const { options, count, total, searchText } = this.state;

        let offset = options.length;

        if (offset < 0) {
            offset = 0;
        }

        if ((offset > 0) && (offset >= count)) {
            this.setState({ more: false });
            return;
        }

        let params = {};
        if (searchText) {
            params = {
                search: searchText
            }
        }

        if (limit) {
            params = {
                offset: offset, limit: limit, ...params
            }
        }

        axios.get(source, {
            params: {
                fields: [ labelKey, valueKey],
                ...params
            }
        })
            .then(response => {
                let { data, meta } = response.data;
                const result = data.map((s) => { return { [valueKey]: s[valueKey], [labelKey]: s[labelKey]} });
                this.setState({
                    options: limit ? options.concat(result) : result,
                    total: meta.total,
                    count: meta.count,
                    //itemsLoading: false,
                    // more: (meta.total > 0)
                }, this.setSelected);
            })


    }

    setSelected() {
        const { value, valueKey, multiple } = this.props;
        const { options } = this.state;

        if (value && !Array.isArray(value)) {
            this.setState({selected: options.find((o) => o[valueKey] === value ) });
        }
    }

    static getDerivedStateFromProps(props, state) {
        // if (props.value !== state.selected) {
        //     return {
        //         selected: props.value
        //     }
        // }
        return null;
    }

    removeValue(value) {
        const { valueKey, multiple, onRemove } = this.props;
        const { selected } = this.state;

        // console.log(selected.findIndex(s => s[valueKey] === value[valueKey]))
        // console.log(selected.splice(selected.findIndex(s => s[valueKey] === value[valueKey]), 1) )

        let selectedValues = '';
        if (multiple) {
            if (selected.length === 1) {
                selectedValues = [];
            } else {
                selectedValues = selected.splice(selected.findIndex(s => s[valueKey] === value[valueKey]), 1)
            }
        }

        this.setState({
            selected: selectedValues
        }, () => onRemove && onRemove(value[valueKey]));


    };

    selectValue({value, option}) {
        const { onSelect, valueKey } = this.props;

        this.setState({
            selected: value
        }, () => onSelect && onSelect(option[valueKey]));

    }

    search(text) {
        const { limit } = this.props;

        limit ? this.remoteSearch(text) : this.localSearch(text);
    }

    remoteSearch(text) {
        this.setState({
            searchText: text,
            options: [],
            total: 0,
            count: 0
        }, this.loadOptions)
    }

    localSearch(text) {
        const { options } = this.state;
        const { labelKey } = this.props;

        // search within local options
        // The line below escapes regular expression special characters:
        // [ \ ^ $ . | ? * + ( )
        const escapedText = text.replace(/[-\\^$*+?.()|[\]{}]/g, "\\$&");
        // Create the regular expression with modified value which
        // handles escaping special characters. Without escaping special
        // characters, errors will appear in the console
        const exp = new RegExp(escapedText, "i");
        this.setState({
            search: text ? options.filter(o => exp.test(o[labelKey])) : []
        });
    }

    render() {
        const { name, label, placeholder, labelKey, valueKey, multiple,
            onSelect, onRemove, value, ...props } = this.props;
        const { options, selected, search } = this.state;

        let s = {};
        if (value) {
            if (!multiple) {
                s[valueKey] = value[0]
            }
        }

        // console.log(multiple)
        // console.log('selected',selected)
        // console.log('value',value)

        return (
            <GrommetSelect name={name} label={label} labelKey={labelKey}
                           options={search.length ? search : options}
                           onMore={this.loadOptions}
                           value={selected || ''} valueKey={valueKey}
                           valueLabel={
                               multiple ?
                                   <Box wrap direction="row">
                                       {selected && selected.length ?
                                           selected.map(v =>
                                               <MultipleSelectedValue key={'opt' + v[valueKey]} labelKey={labelKey} value={v}
                                                                      onRemove={this.removeValue} />
                                           ) : <Placeholder placeholder={placeholder} />

                                       }
                                   </Box> :
                                   <Box>
                                       {selected ?
                                           <SelectedValue value={selected} labelKey={labelKey} onRemove={this.removeValue} /> :
                                           <Placeholder placeholder={placeholder} />
                                       }
                                   </Box>
                           }
                           onSearch={this.search}
                           multiple={multiple}
                           onChange={this.selectValue}
                           emptySearchMessage=""
                           searchPlaceholder=""
                           {...props}
            >
                {(option, index, options, state) =>
                    <Box pad="small" background={state.selected ? 'neutral-3' : ''}>{option[labelKey]}</Box>
                }
            </GrommetSelect>
        );


    }
}

Select.defaultProps = {
    labelKey: 'name',
    valueKey: 'id'
};

export default Select;