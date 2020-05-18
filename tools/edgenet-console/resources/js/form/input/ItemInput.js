import React from "react";
import axios from "axios";
import {Layer, Box, Stack, Button} from "grommet";
import { FormClose } from "grommet-icons";

import {Data} from "../../data";
import {List} from "../../data/views";

const Item = ({item}) =>
    <Box pad="small">
        {item.name || item.title}
    </Box>

const Items = ({resource, onClose, onClick}) =>
    <Layer position="center" modal animate={false} onEsc={onClose}
           onClickOutside={onClose}>
        <Box pad="small" width="large" height="large">
            <Data url={"/api/" + resource}>
                <List onClick={onClick}>
                    <Item />
                </List>
            </Data>
        </Box>
    </Layer>


class ItemInput extends React.Component {

    constructor(props) {
        super(props);
        this.state = {
            item: null,
            dialog: false,
        };

        this.toggleDialog = this.toggleDialog.bind(this);
        this.onSelect = this.onSelect.bind(this);
        this.onClear = this.onClear.bind(this);
    }

    componentDidMount() {
        const { value, resource } = this.props;

        if (value) {
            axios.get('/api/' + resource + '/' + value)
                .then(({data}) => this.setState({item: data}))
                .catch(err => console.log(err))
        }
    }

    toggleDialog() {
        this.setState({dialog: !this.state.dialog})
    }

    onSelect(value) {
        const { onChange } = this.props;

        if (onChange) {
            this.setState({
                item: value,
                dialog: false
            }, () => onChange({target: {value: value.id}}))
        }
    }

    onClear() {
        const { onChange } = this.props;

        this.setState({
            item: null,
        }, () => onChange({target: {value: null}}))
    }

    render() {
        const { name, value, resource, placeholder } = this.props;
        const { item, dialog } = this.state;

        if (dialog)
        return <Items resource={resource}
                          onClose={this.toggleDialog}
                          onClick={this.onSelect} />

        return (
            <Stack anchor="right">
                <Box pad="small" onClick={this.toggleDialog}>
                    {item ? item.name || item.title : placeholder}
                </Box>
                <Box background="light-1">
                    <Button icon={<FormClose />} onClick={this.onClear} />
                </Box>
            </Stack>
        );

    }

}

export default ItemInput;
