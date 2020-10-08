import React, {useEffect, useContext, useState} from "react";
import axios from "axios";

import { ConsoleContext } from "../../index";
import { AuthenticationContext } from "../../authentication";
import {Box, Text, Button} from "grommet";
import {StatusGood, StatusDisabled, Validate} from "grommet-icons";

import User from "../components/User";

const Slice = ({resource, approveUser}) =>
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
                {/*<Button label="Approve" icon={<Validate />} onClick={() => approveUser(resource.metadata)} />*/}
            </Box>
        </Box>
    </Box>;



const Slices = () => {
    const [ resources, setResources ] = useState([]);
    const [ loading, setLoading ] = useState(false);
    const { config } = useContext(ConsoleContext);
    const { user } = useContext(AuthenticationContext);

    console.log(user)
    useEffect(() => {
        loadRequests();
    }, [])

    const loadRequests = () => {
        axios.get('/apis/apps.edgenet.io/v1alpha/namespaces/authority-' + user.authority + '/slices', {
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

    const createSlice = (user) => {
        axios.patch(
            '/apis/apps.edgenet.io/v1alpha/namespaces/' + user.namespace + '/userregistrationrequests/' + user.name,
            [{ op: 'replace', path: '/spec/approved', value: true }],
            { headers: { 'Content-Type': 'application/json-patch+json' } }
        )
            .then(res => {
                loadRequests()
                console.log(res)
            })
            .catch(err => console.log(err.message))
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
        <Box overflow="auto" pad="medium">

            {resources.map(resource =>
                <Box key={resource.metadata.name}
                    // onMouseEnter={() => this.setState({ isMouseOver: true })}
                    //  onMouseLeave={() => this.setState({ isMouseOver: false })}
                    //  onClick={() => onClick(item)}
                    //  background={background}
                     border={{side:'bottom', color:'light-4'}} flex={false}>
                    <Slice resource={resource} />
                </Box>
            )}
        </Box>

    )


}


export default Slices;