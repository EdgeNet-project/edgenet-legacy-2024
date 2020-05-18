import React from 'react';
import {Box, Anchor, Text, Image } from "grommet";
import moment from "moment";


const filesize = (size) => size;

const Image = ({item}) =>
    <Box direction="row" gap="small" pad="small" flex="grow">
        <Box height="xsmall" width="xsmall">
            <Image src={'/img/thumbnail-' + item.id + '.jpg'} fit="cover" title={item.title} />
        </Box>
        <Box flex="grow">
            {item.title && <Text size="small">{item.title}</Text>}
            <Text size="small">
                <Anchor className="small" href={"/doc/" + item.id + "-" + item.name }
                        icon={<Download size="small"/>} label={item.name} />
            </Text>

            <Box>
                <Text size="small">crée le {moment(item.created_at).format('lll')}</Text>
                {/*<Text size="small">modifié le {moment(item.updated_at).format('lll')}</Text>*/}
            </Box>
            <Text size="small">{filesize(item.size)}</Text>
        </Box>
        <Box align="end">
            {/*<ButtonDelete label={false} id={item.id} />*/}
        </Box>

    </Box>;

const ImagesList = () =>
    <div></div>;

export default ImagesList;
