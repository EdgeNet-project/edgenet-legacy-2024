import React from "react";
import LocalizedStrings from "react-localization";
import { Box, Button, Layer } from "grommet";
import { Trash } from "grommet-icons";
import { DataContext } from "../../data/Data";

const strings = new LocalizedStrings({
    en: {
        delete: "Delete",
        cancel: "Cancel",
        confirmDelete: "Are you sure you want to delete this item?"
    },
    fr: {
        delete: "Supprimer",
        cancel: "Anuller",
        confirmDelete: "Êtes vous sûr de vouloir supprimer définitivement cet objet ?"
    }
});

class ButtonDelete extends React.Component {

    constructor(props) {
        super(props);
        this.state = {
            confirmDelete: false
        };

        this.cancel = this.cancel.bind(this);
        this.confirm = this.confirm.bind(this);
    }

    cancel() {
        this.setState({confirmDelete: false});
    }

    confirm() {
        this.setState({confirmDelete: true});
    }

    render() {
        const { label, id } = this.props;
        const { confirmDelete } = this.state;
        const { deleteItem } = this.context;

        if (!id) return null;

        return (
                <Box pad="small">
                    <Button icon={<Trash color="status-critical" />} plain
                            color="status-critical"
                            onClick={this.confirm}
                            label={label === undefined ? strings.delete : label} />
                    {confirmDelete &&
                    <Layer onEsc={this.cancel} onClickOutside={this.cancel}>
                        <Box pad="medium">
                            {strings.confirmDelete}
                        </Box>
                        <Box direction="row" justify="center" gap="small" pad="medium" alignContent="center">
                            <Button label={strings.cancel} onClick={this.cancel}/>
                            <Button primary label={strings.delete} onClick={() => deleteItem(id)}/>
                        </Box>
                    </Layer>
                    }
                </Box>
        );
    }
}

ButtonDelete.contextType = DataSourceContext;

export default ButtonDelete;