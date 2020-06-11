import React from "react";
import { Box, Button, Text } from "grommet";
import { FilterableConsumer } from "./Filterable";

class SelectOption extends React.Component {

    constructor(props) {
        super(props);
        this.state = {
            hover: false
        }
    }

    render() {
        let { hover } = this.state;
        let { label, icon, active, onClick } = this.props;

        return (
            <Box background={active ? "brand" : (hover ? "light-3" : null)} round="small" pad="xsmall"
                 onMouseEnter={() => this.setState({hover: true})} onMouseLeave={() => this.setState({hover: false})}
                 onClick={onClick}>
                <Button icon={icon} label={label} plain />
            </Box>
        );
    }
}

const SelectFilter = ({ name, label, multi, options }) =>
    <FilterableConsumer>
        {
            ({hasFilter, addFilter, removeFilter}) =>
                <Box>
                    <Text size="small" margin={{bottom: 'xsmall'}}>{label}</Text>
                    <Box direction="row" gap="small">
                        {options.map(({value, label, icon}) => {
                                let active = hasFilter(name, value);

                                return <SelectOption key={label} label={label} icon={icon} active={active}
                                                     onClick={active ?
                                                         () => removeFilter(name, multi ? value : null) :
                                                         () => addFilter(name, value, multi)} />
                            })
                        }
                    </Box>
                </Box>
        }
    </FilterableConsumer>;

export default SelectFilter;
