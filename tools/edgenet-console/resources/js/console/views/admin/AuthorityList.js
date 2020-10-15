import React, {useEffect, useContext, useState} from "react";
import axios from "axios";

import { ConsoleContext } from "../../index";
import {Box, Text, Button} from "grommet";
import {StatusGood, StatusDisabled, Validate} from "grommet-icons";

import { Authority, AuthorityAddress, AuthorityContact } from "../../resources";

const AuthorityRow = ({resource}) =>
    <Box pad="small" direction="row" justify="between">
        <Box gap="small">
            <Authority resource={resource} />

            <Box gap="medium" direction="row">
                <Text size="small" >
                    <i>Address</i>
                    <AuthorityAddress resource={resource} />
                </Text>

                <Text size="small">
                    <i>Contact</i>
                    <AuthorityContact resource={resource} />
                </Text>
            </Box>
        </Box>

        <Box justify="between">
            <Box align="end">
                {resource.spec.enabled ? <StatusGood color="status-ok" /> : <StatusDisabled />}
            </Box>
            <Box align="end">
                <Text size="small">
                    UID: {resource.metadata.uid} <br />
                    Name: {resource.metadata.name}
                </Text>
            </Box>
        </Box>
    </Box>;



const AuthorityList = () => {
    const [ resources, setResources ] = useState([]);
    const [ loading, setLoading ] = useState(false);
    const { config } = useContext(ConsoleContext);

    useEffect(() => {
        loadRequests();
    }, [])

    const loadRequests = () => {
        axios.get('/apis/apps.edgenet.io/v1alpha/authorities', {
            // params: { ...queryParams, page: current_page + 1 },
            // paramsSerializer: qs.stringify,
        })
            .then(({data}) => {
                if (data.items) {
                    setResources(data.items);
                }
                // this.setState({
                //     ...data, loading: false
                // });
            })
            .catch(error => {
                console.log(error)
            });
    }


    // if (loading) {
    //     return <Box>Loading</Box>;
    // }
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
                    <AuthorityRow resource={resource}  />
                </Box>
            )}
        </Box>

    )


}


export default AuthorityList;