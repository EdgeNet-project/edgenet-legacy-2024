import React, {useEffect, useContext, useState} from "react";
import axios from "axios";

import { AuthenticationContext } from "../../authentication";
import { Box } from "grommet";

import {
    Node,
} from "../../resources";


const NodeRow = ({resource}) =>
    <Box pad="small" direction="row" justify="between">
        <Box gap="small">
            <Node resource={resource} />
        </Box>
    </Box>;

const NodeList = () => {
    const [ resources, setResources ] = useState([]);
    const [ loading, setLoading ] = useState(false);
    const { user } = useContext(AuthenticationContext);

    useEffect(() => {
        loadResources();
    }, [])

    const loadResources = () => {
        console.log(user)
        axios.get('/api/v1/nodes', {
            // params: { ...queryParams, page: current_page + 1 },
            // paramsSerializer: qs.stringify,
        })
            .then(({data}) => {
                setResources(data.items);
                // this.setState({
                //     ...data, loading: false
                // });
            })
            .catch(error => {
                console.log(error)
            });
    }

    if (loading) {
        return <Box>Loading</Box>;
    }


    return (

        <Box overflow="auto" pad="small">

            {resources.map(resource =>
                <Box key={resource.metadata.name}
                    // onMouseEnter={() => this.setState({ isMouseOver: true })}
                    //  onMouseLeave={() => this.setState({ isMouseOver: false })}
                    //  onClick={() => onClick(item)}
                    //  background={background}
                     border={{side:'bottom', color:'light-3'}}
                     flex={false}>
                    <NodeRow resource={resource}  />
                </Box>
            )}
        </Box>
    )


}


export default NodeList;