import React from 'react';
import axios from "axios";
import {Layer, Stack, Box, Text, Image, Button, Meter } from "grommet";
import {Add, Trash, Save} from "grommet-icons";
import moment from "moment";
import fileSize from "filesize";

import { List } from "../../data/views";
import { Data, DataContext } from "../../data";


const ImageList = ({item}) =>
    <Box direction="row" gap="small" pad="small" flex="grow">
        <Box height="xsmall" width="xsmall">
            <Image src={item.thumb} fit="cover" title={item.title} />
        </Box>
        <Box flex="grow">
            {item.title && <Text size="small">{item.title}</Text>}
            {/*<Text size="small">*/}
            {/*    <Anchor className="small" href={"/doc/" + item.id + "-" + item.name }*/}
            {/*            icon={<Download size="small"/>} label={item.name} />*/}
            {/*</Text>*/}

            <Box>
                {/*<Text size="small">crée le {moment(item.created_at).format('lll')}</Text>*/}
                {/*<Text size="small">modifié le {moment(item.updated_at).format('lll')}</Text>*/}
            </Box>
            <Text size="small">{item.name}</Text>
            <Text size="small">{fileSize(item.size)}</Text>
        </Box>
        <Box align="end">
            {/*<ButtonDelete label={false} id={item.id} />*/}
        </Box>
    </Box>;

class ImagePreview extends React.Component {

    constructor(props) {
        super(props);
        this.state = {
            preview: null
        }

        this.reader = new FileReader();
    }

    componentDidMount() {
        const { file } = this.props;

        this.reader.onloadend = () => this.setState({
            preview: this.reader.result,
            file: file
        });

        this.reader.readAsDataURL(file);

        this.onClose = this.onClose.bind(this);
    }

    onClose() {
        const { onClose } = this.props;
        this.setState({
            preview: null
        }, onClose);
    }

    render() {
        const { file, onSave, onDelete, progress} = this.props;
        const { preview } = this.state;

        return (
            <Layer position="center" modal animate={false} onEsc={this.onClose}
                   onClickOutside={this.onClose}>
                <Stack fill={true}>
                    <Box height="large">
                        <Image src={preview} fit="contain" fill="horizontal" title={file.name} />
                    </Box>
                    {progress > 0 && <Box align="center" justify="center" fill background={{
                        'color': 'dark-1',
                        'opacity': 'strong',
                    }}>
                        <Text>Uploading, please wait...</Text>
                        <Meter values={[{'value': progress}]} />
                    </Box>}
                </Stack>
                <Box pad="small" grow="2">
                    <Text size="small">Filename: {file.name}</Text>
                    <Text size="small">Size: {fileSize(file.size)}</Text>

                    <Box justify="end" direction="row" gap="small">
                        {onDelete && <Button plain label="Effacer" icon={<Trash color="status-critical" />} onClick={onDelete} />}
                        <Button primary disabled={progress > 0} label="Sauvgarder" icon={<Save />} onClick={onSave} />
                    </Box>
                </Box>
            </Layer>
        );
    }
}


class UploadImage extends React.Component {

    constructor(props) {
        super(props);
        this.state = {
            file: null,

            progress: 0,
            message: ''
        };

        this.selectImage = this.selectImage.bind(this);
        this.upload = this.upload.bind(this);
    }

    /*
        Called after the system dialog to select an image
        TODO: properties are name, size, type
    */
    selectImage(event) {
        if (event.target.files && event.target.files[0]) {
            this.setState({
                file: event.target.files[0]
            });
        }
    }

    upload() {
        const { file } = this.state;
        const { name } = this.props;
        const { url, pushItem } = this.context;

        let data = new FormData();
        data.append(name, file);

        const config = {
            onUploadProgress: (ev) => this.setState({progress: Math.round( (ev.loaded * 100) / ev.total ) })
        }

        axios.post(url, data, config)
            .then(({data}) => {
                pushItem(data.data)
                this.setState({progress: 0, file: null})
            })
            .catch(err => this.setState({progress: 0, message: err.message}));
    }

    render() {
        const { name } = this.props;
        const { file, progress, message } = this.state;

        return (
            <Box pad={{vertical: 'small'}}>
                <Box>
                    <Button icon={<Add/>} label="Sélectionner une image" plain={true}
                            onClick={() => this.inputElement.click()} />
                    <input type="file" onChange={this.selectImage} ref={(ref) => { this.inputElement = ref }}
                           id={name} name={name} hidden={true} />
                </Box>
                {message}
                {file && <ImagePreview file={file} progress={progress} onSave={this.upload} onClose={() => this.setState({file:null})} />}
            </Box>
        )
    }
}

UploadImage.contextType = DataContext;

class ModifyImage extends React.Component {

    constructor(props) {
        super(props);
        this.state = {
            image: null,
            preview: null,
            confirmDelete: false,

        }

        this.handleClose = this.handleClose.bind(this);
        this.handleDelete = this.handleDelete.bind(this);
    }

    componentDidMount() {
        const { image } = this.props;

        axios.get(image.url, {responseType: 'blob'})
            .then(({data}) => this.setState({
                image: image,
                preview: new Blob([data])
            }))
            .catch(err => console.log(err))
    }

    handleClose() {
        const { onClose } = this.props;
        this.setState({
            preview:null
        }, onClose);
    }

    handleDelete() {
        const { image } = this.state;
        const { url, pullItem } = this.context;

        if (!image && !image.id) return;

        axios.delete(url + "/" + image.id)
            .then(() => pullItem(image))
            .then(this.handleClose)
    }

    render() {
        const { preview } = this.state;

        return (
            preview && <ImagePreview file={preview} onDelete={this.handleDelete} onClose={this.handleClose} />
        );
    }
}

ModifyImage.contextType = DataContext;


class Images extends React.Component {
    constructor(props) {
        super(props);
        this.state = {
            image: null
        }

        this.setImage = this.setImage.bind(this);
        this.unsetImage = this.unsetImage.bind(this);

    }

    setImage(image) {
        this.setState({image: image})
    }
    unsetImage() {
        this.setState({image: null})
    }


    render() {
        const { resource, id } = this.props;
        const { image } = this.state;

        return (
            <Data orderable url={"/api/" + resource + "/" + id + "/images"}>
                <List onClick={this.setImage}>
                    <ImageList />
                </List>
                <UploadImage name="image" />
                {image && <ModifyImage image={image}  onClose={this.unsetImage} />}
            </Data>
        )
    }
}


Images.title = 'Images';

export default Images;
