import React, {useContext, useState, useEffect} from "react";
import axios from "axios";

import { ConsoleContext } from "../../index";
import { Box, Text } from "grommet";

const Secrets = () => {
    const [ secrets, setSecrets ] = useState([]);
    const [ loading, setLoading ] = useState(false);
    const { config } = useContext(ConsoleContext);

    useEffect(() => {
        getSecrets()
    }, [])

    const getSecrets = () => {
        // const { items, current_page, last_page, queryParams } = this.state;

        // if (!api) return false;

        // if (current_page >= last_page) return;

        //const url = '';
        axios.get('/api/v1/secrets', {
            // params: { ...queryParams, page: current_page + 1 },
            // paramsSerializer: qs.stringify,
        })
            .then(({data}) => {
                console.log(data)
                setSecrets(data.items);
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
        <Box overflow="auto" pad={{horizontal:'medium'}}>
            {
                secrets.map(secret =>
                    <Box key={secret.metadata.name} flex={false} pad={{vertical:'small'}} border={{side:'bottom',color:'light-2'}}>
                        {secret.metadata.name}
                        <Text size="small">Namespace: {secret.metadata.namespace}</Text>
                    </Box>
                )
            }
        </Box>

    )


}


export default Secrets;