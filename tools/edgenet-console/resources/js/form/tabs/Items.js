import React from "react";
import {Box} from "grommet";
import {Data} from "../../data";
import {List} from "../../data/views";


const Item = ({item}) =>
    <Box>
        {item.name}
        {item.title}
    </Box>

const Items = ({resource, id, related}) =>
    <Data orderable url={"/api/" + resource + "/" + id + "/" + related}>
        <List>
            <Item />
        </List>
        {/*<UploadButton name="image" url={"/api/" + resource + "/" + id + "/images"} />*/}
    </Data>;

Items.title = 'Items';

export default Items;
