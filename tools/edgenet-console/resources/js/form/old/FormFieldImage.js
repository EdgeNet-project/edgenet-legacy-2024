import React from 'react';

import {Box, Text, Anchor, Button, Image} from "grommet";
import { Add } from "grommet-icons";
import { DataSourceContext } from "../DataSource";
import filesize from "filesize";


const ImageField = ({file, preview, onClick}) =>
    <Box direction="row" gap="small">
        <Anchor onClick={onClick}>
            <Box width="small" height="small">
            <Image src={preview} fit="cover" title={file.name} />
            </Box>
        </Anchor>
        <Box>
            <Text size="small"><Anchor onClick={onClick}>{file.name}</Anchor></Text>
            <Text size="small">{filesize(file.size)}</Text>
        </Box>
    </Box>;

class FormFieldImage extends React.Component {

    constructor(props) {
        super(props);
        this.state = {
            file: null,
            preview: null,
        };

        this.selectImage = this.selectImage.bind(this);
    }

    /*
        Called after the system dialog to select an image
        TODO: properties are name, size, type
    */
    selectImage(event) {
        const { attachFile } = this.context;

        let file = null;
        if (event.target.files && event.target.files[0]) {
            file = event.target.files[0];
            const reader = new FileReader();
            reader.onloadend = () => this.setState({
                preview: reader.result,
                file: file
            }, () => attachFile(file));

            reader.readAsDataURL(file);

        }
    }

    render() {
        const { name } = this.props;
        const { file, preview } = this.state;

        return (
            <Box pad={{vertical: 'small'}}>
                {file ?
                    <ImageField file={file} preview={preview} onClick={() => this.inputElement.click()}/> :
                    <Button icon={<Add/>} label="Sélectionner une image" plain={true}
                            onClick={() => this.inputElement.click()}/>
                }
                <input type="file"
                       onChange={this.selectImage}
                       ref={(ref) => { this.inputElement = ref }}
                       id={name} name={name} hidden={true}
                />
            </Box>
        )
    }

}

FormFieldImage.contextType = DataSourceContext;

export default FormFieldImage;