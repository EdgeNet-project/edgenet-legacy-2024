import React from 'react';

import { Box, Text, Anchor, Button } from "grommet";
import { Add, DocumentPdf, DocumentExcel, DocumentWord, DocumentZip, DocumentPpt,
    DocumentRtf, DocumentText, DocumentCsv, Document } from "grommet-icons";
import { DataSourceContext } from "../DataSource";
import filesize from "filesize";
import mime from "mime-types";
import moment from "moment";

const DocumentIcon = ({mime_type}) => {
    let type = mime.extension(mime_type);
    switch(type) {
        case 'pdf':
            return <DocumentPdf />;
        case 'doc':
        case 'docx':
            return <DocumentWord />;
        case 'ppt':
            return <DocumentPpt />;
        case 'xls':
            return <DocumentExcel />;
        case 'rtf':
            return <DocumentRtf />;
        case 'txt':
        case 'text':
            return <DocumentText />;
        case 'csv':
            return <DocumentCsv />;
        case 'zip':
            return <DocumentZip />;
        default:
            return <Document />;
    }
};

const DocumentField = ({file, onClick}) =>
    <Box direction="row" gap="small">
        <Box>
            <Anchor onClick={onClick}><DocumentIcon mime_type={file.mime_type} /></Anchor>
            {/*{doc.idx !== null && <Anchor className="small" href={"/doc/" + doc.id + "-" + doc.name }><DownloadIcon size="xsmall"/>Télecharger</Anchor>}*/}
        </Box>
        <Box>
            <Text size="small"><Anchor onClick={onClick}>{file.name}</Anchor></Text>
            <Text size="small">{filesize(file.size)}</Text>
        </Box>
    </Box>;

class FormFieldDocument extends React.Component {

    constructor(props) {
        super(props);
        this.state = {
            file: null
        };

        this.selectDocument = this.selectDocument.bind(this);
    }

    /*
        Called after the system dialog to select an image
        TODO: properties are name, size, type
    */
    selectDocument(event) {
        const { attachFile } = this.context;

        let file = null;
        if (event.target.files && event.target.files[0]) {
            file = event.target.files[0];
            this.setState({
                file: file
            }, () => attachFile(file));
        }
    }

    render() {
        const { name } = this.props;
        const { file } = this.state;

        return (
            <Box pad={{vertical: 'small'}}>
                {file ?
                    <DocumentField file={file} onClick={() => this.inputElement.click()}/> :
                    <Button icon={<Add/>} label="Sélectionner un document" plain={true}
                            onClick={() => this.inputElement.click()}/>
                }
                <input type="file"
                       onChange={this.selectDocument}
                       ref={(ref) => { this.inputElement = ref }}
                       id={name} name={name} hidden={true}
                />
            </Box>
        )
    }

}

FormFieldDocument.contextType = DataSourceContext;

export default FormFieldDocument;