import React, {useEffect, useContext, useState} from "react";
import { useParams } from "react-router-dom"
import axios from "axios";

import { ConsoleContext } from "../../index";
import { AuthenticationContext } from "../../authentication";
import { Box, InfiniteScroll } from "grommet";

import {
    Slice,
} from "../../resources";


const SliceRow = ({resource}) =>
    <Box pad="small" direction="row" justify="between">
        <Box gap="small">
            <Slice resource={resource} />
        </Box>
    </Box>;

const SliceList = () => {
    const [ resources, setResources ] = useState([]);
    const [ loading, setLoading ] = useState(false);
    const { user } = useContext(AuthenticationContext);

    useEffect(() => {
        loadResources();
    }, [])

    const loadResources = () => {
        console.log(user)
        axios.get('/apis/apps.edgenet.io/v1alpha/slices', {
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
    //
    // return (
    //     <Box overflow="auto">
    //         {
    //             resources.map(resource => <NodeList resource={resource} /> )
    //         }
    //     </Box>
    // )

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
                    <SliceRow resource={resource}  />
                </Box>
            )}
        </Box>

    )


}


export default SliceList;