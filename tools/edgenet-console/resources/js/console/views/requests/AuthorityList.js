import React, {useEffect, useContext, useState} from "react";
import axios from "axios";

import { ConsoleContext } from "../../index";
import {Box, Text, Button} from "grommet";
import {StatusGood, StatusDisabled, Validate} from "grommet-icons";

import { Authority, AuthorityAddress, AuthorityContact } from "../../resources";

const AuthorityRequest = ({resource, approveAuthority}) =>
    <Box pad="small" direction="row" justify="between">
        <Box gap="small">
            <Authority resource={resource} />
            <Text size="small">
                UID: {resource.metadata.uid} <br />
                Name: {resource.metadata.name}
            </Text>

            <Text size="small" >
                <i>Address</i>
                <AuthorityAddress resource={resource} />
            </Text>

            <Text size="small">
                <i>Contact</i>
                <AuthorityContact resource={resource} />
            </Text>
        </Box>

        <Box justify="between">
            <Box>
                <Text size="small">
                    E-Mail Verified: {resource.status.emailverified ? <StatusGood size="small" color="status-ok" /> : <StatusDisabled size="small" />}
                </Text>
                <Text size="small">
                    Request expires: {resource.status.expires}
                </Text>
                <Text size="small">
                    Messages: {resource.status.message.map((m, j) => <Text size="small" key={"m-" + j + resource.metadata.name}>{m}</Text>)}
                </Text>
            </Box>
            <Box align="end">
                <Button label="Approve" icon={<Validate />} onClick={() => approveAuthority(resource.metadata.name)} />
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
        axios.get('/apis/apps.edgenet.io/v1alpha/authorityrequests', {
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

    const approveAuthority = (authority) => {
        axios.patch(
            '/apis/apps.edgenet.io/v1alpha/authorityrequests/' + authority,
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
        <Box overflow="auto" pad="small">

                {resources.map(resource =>
                   <Box key={resource.metadata.name}
                        // onMouseEnter={() => this.setState({ isMouseOver: true })}
                        //  onMouseLeave={() => this.setState({ isMouseOver: false })}
                        //  onClick={() => onClick(item)}
                        //  background={background}
                        border={{side:'bottom', color:'light-4'}}
                        flex={false}>
                        <AuthorityRequest resource={resource} approveAuthority={approveAuthority} />
                    </Box>
                )}
        </Box>

    )


}


export default AuthorityList;