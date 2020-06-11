import React from "react";
import axios from "axios";
import Select from 'react-select';

class Select extends React.Component {

    constructor(props) {
        super(props);
        this.state = {
            options: []
        };

        this.handleChange = this.handleChange.bind(this);
    }

    componentDidMount() {
        const { source, labelKey, valueKey } = this.props;

        axios.get(source)
            .then(({data}) =>
                this.setState({
                    options: data.data.map((d) => {
                        return {
                            value: d[valueKey], label: d[labelKey]
                        }
                    }),
                })
            );
    }

    handleChange({value}) {
        const { onChange, valueKey } = this.props;

        if (onChange) {
            onChange({target: {value: { [valueKey]: value }}})
        }
    }

    render() {
        const { options } = this.state;
        const { name, value, placeholder, valueKey } = this.props

        if (!options) return null;

        return <Select className="ReactSelect"
                       placeholder={placeholder}
                       isSearchable={true}
                       isClearable={true}
                       options={options}
                       value={value ? options.find(o => value[valueKey] === o.value) : null}
                       name={name}
                       onChange={this.handleChange} />;
    }
}

export default Select;
