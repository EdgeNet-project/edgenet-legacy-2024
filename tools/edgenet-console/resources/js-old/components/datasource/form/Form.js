import React from "react";
import PropTypes from "prop-types";
import { Box, Button, Form as GrommetForm, Heading, Layer } from "grommet";
import { Close } from "grommet-icons";

import { DataSourceContext } from "../DataSource";



class Form extends React.Component {

    constructor(props) {
        super(props);
        this.state = {
            displayDialog: false
        };

        this.showDialog = this.showDialog.bind(this);
        this.hideDialog = this.hideDialog.bind(this);
        this.submit = this.submit.bind(this);
    }

    showDialog() {
        this.setState({displayDialog: true})
    }

    hideDialog() {
        this.setState({displayDialog: false})
    }

    submit({value}) {
        const { saveItem } = this.context;

        saveItem(value, this.hideDialog());
    }

    render() {
        const { displayDialog } = this.state;
        const { title, label, icon, dialog } = this.props;
        const { item, changeItem, resetItem } = this.context;

        let form = <GrommetForm value={item || {}}
                                onReset={resetItem} onChange={changeItem}
                                onSubmit={this.submit}>
            {this.props.children}
        </GrommetForm>;

        if (dialog) {
            form = (
                <Box>
                    <Box pad="small" justify="start">
                        <Button plain label={label ? label : title}
                                icon={icon} onClick={this.showDialog}
                                alignSelf="start"
                        />
                    </Box>
                    {displayDialog && <Layer position="center" modal onEsc={this.hideDialog}
                                             onClickOutside={this.hideDialog}>
                        <Box gap="medium" pad="medium" width="medium">
                            <Box direction="row" flex="grow" justify="between">
                                {title && <Heading level={3} margin="none">{title}</Heading>}
                                <Button plain icon={<Close />} onClick={this.hideDialog} />
                            </Box>
                            {form}
                        </Box>
                    </Layer>}
                </Box>
            );
        }

        return form;

    }
}

Form.contextType = DataSourceContext;

Form.defaultProps = {
};

export default Form;