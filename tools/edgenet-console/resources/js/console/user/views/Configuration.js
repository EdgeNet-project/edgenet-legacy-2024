import React, { useState, useContext, useEffect, useRef } from "react";
import axios from "axios";
import {Box, Heading, Text, Button, TextArea} from "grommet";
import {Download, Copy, Down} from "grommet-icons";
import { AuthenticationContext } from "../../authentication";

const Code = ({children}) =>
    <Text size="small">
        <pre style={{backgroundColor:'#ededed',padding:'20px 0'}}>
            {children}
        </pre>
    </Text>;

const Configuration = () => {
    const { user, edgenet } = useContext(AuthenticationContext);
    const [ cluster, setCluster ] = useState({})
    const textareaEl = useRef(null);

    useEffect(() => {
        getClusterInfo()
    }, [])

    const getClusterInfo = () =>
        axios.get('/api/cluster', {
            // params: { ...queryParams, page: current_page + 1 },
            // paramsSerializer: qs.stringify,
        })
            .then(({data}) => {
                console.log(data)
                setCluster(data);
                // this.setState({
                //     ...data, loading: false
                // });
            })
            .catch(error => {
                console.log(error)
            });

    const copyToClipboard = () => {
        // textareaEl.current.focus();
        textareaEl.current.select()
        document.execCommand("copy")
    }

    const downloadConfig = () => {
        const element = document.createElement("a");
        const file = new Blob([config], {type: 'text/yaml'});
        element.href = URL.createObjectURL(file);
        element.download = "edgenet-" + user.authority + "-" + user.name + ".yml";
        document.body.appendChild(element); // Required for this to work in FireFox
        element.click();
    }

const config = `apiVersion: v1
kind: Config
clusters:
- name: edgenet-cluster
  cluster:
    certificate-authority-data: ` + cluster.ca + `
    server: ` + cluster.server + `
contexts:
- name: edgenet
  context:
    cluster: edgenet-cluster
    namespace: authority-` + user.authority + `
    user: ` + user.email + `
current-context: edgenet
users:
- name: ` + user.email + `
  user:
    token: ` + user.api_token;

    return (
        <Box pad="medium">
            <Heading size="small">
                Your EdgeNet configuration
            </Heading>

            <Box direction="row" gap="medium" margin={{vertical:'medium'}} justify="end">
                <Button plain label="Download" icon={<Download />} onClick={downloadConfig} />
                <Button plain label="Copy to clipboard" icon={<Copy />} onClick={copyToClipboard} />
            </Box>

            <TextArea ref={textareaEl} rows="10" value={config} />

            <Box margin={{vertical:'medium'}} pad={{vertical:'medium'}} border={{side:'top',color:'light-5'}}>
                Copy your configuration file to edgenet-{user.authority}-{user.name}.yml"

                <Heading level="3">
                    Using kubectl with your configuration file
                </Heading>
                <Text>
                    To use EdgeNet your configuraton file copy it on your computer:
                </Text>
                <Code>
                    # On Linux and Macos kubernetes config files are stored in your home
                    cp edgenet-{user.authority}-{user.name}.yml $HOME/.kube
                </Code>

                <Heading level="3">
                    Merging your configurations files
                </Heading>
                <Text>
                    kubeconfig files are structured YAML files, you can use kubectl to merge your configuraton files:
                </Text>
                <Text size="small">
                    <pre style={{backgroundColor:'#ededed',padding:'20px 0'}}>
                        cp $HOME/.kube/config $HOME/.kube/config.backup.$(date +%Y-%m-%d.%H:%M:%S)
                    </pre>
                </Text>
            </Box>

        </Box>
    );
}

export default Configuration;