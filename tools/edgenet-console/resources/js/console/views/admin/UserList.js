import React, {useEffect, useContext, useState} from "react";
import axios from "axios";

import { ConsoleContext } from "../../index";
import {Box, Text, Button} from "grommet";
import {StatusGood, StatusDisabled, Validate} from "grommet-icons";

import User from "../../resources/components/User";

const UserRow = ({resource}) =>
    <Box pad="small" direction="row" justify="between">
        <Box gap="small">
            <User resource={resource} />
            <Text size="small">
                UID: {resource.metadata.uid} <br />
                Name: {resource.metadata.name} <br />
                Namespace: {resource.metadata.namespace}
            </Text>

        </Box>

        <Box justify="between">
            <Box>

            </Box>
            <Box align="end">
            </Box>
        </Box>
    </Box>;



const UserList = () => {
    const [ resources, setResources ] = useState([]);
    const [ loading, setLoading ] = useState(false);
    const { config } = useContext(ConsoleContext);

    useEffect(() => {
        loadRequests();
    }, [])

    const loadRequests = () => {
        axios.get('/apis/apps.edgenet.io/v1alpha/users', {
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
                    <UserRow loading={loading} resource={resource} />
                </Box>
            )}
        </Box>

    )


}


export default UserList;