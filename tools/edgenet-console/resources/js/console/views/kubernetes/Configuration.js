import React, { useState, useContext, useEffect, useRef } from "react";
import axios from "axios";
import {Box, Heading, Text, Button, Anchor, TextArea} from "grommet";
import {Download, Copy} from "grommet-icons";
import { AuthenticationContext } from "../../authentication";

const Code = ({children}) =>
    <Text size="small">
        <pre style={{backgroundColor:'#ededed',padding:'20px 10px',overflow:'auto'}}>
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
            <Heading size="small" margin="none">
                Your EdgeNet configuration
            </Heading>

            <Box direction="row" gap="medium" margin={{vertical:'medium'}} justify="end">
                <Button plain label="Download" icon={<Download />} onClick={downloadConfig} />
                <Button plain label="Copy to clipboard" icon={<Copy />} onClick={copyToClipboard} />
            </Box>

            <TextArea ref={textareaEl} rows="10" value={config} />

            <Box border={{side:'top',color:'light-5'}} margin={{top:'medium'}}>

                <Heading level="3">
                    Using kubectl with your configuration file
                </Heading>
                <Text>
                    To use EdgeNet download your configuraton file on your computer:
                </Text>
                <Code>
                    # On Linux and Macos kubernetes config files are stored in your home <br/>
                    mv edgenet-{user.authority}-{user.name}.yml $HOME/.kube
                </Code>
                <Text>
                    Specify the configuration file path when using kubectl:
                </Text>
                <Code>
                    kubectl get nodes --kubeconfig=$HOME/.kube/edgenet-{user.authority}-{user.name}.yml
                </Code>

                <Heading level="3">
                    Merging your configurations files
                </Heading>
                <Text>
                    kubeconfig files are structured YAML files and $HOME/.kube/config is the default configuration file. <br />
                    If you already have one or more cluster configurations you can use kubectl to merge your edgenet configuraton files. <br />
                    As a first step make a backup of your current default configuration:
                </Text>
                <Code>
                    cp $HOME/.kube/config $HOME/.kube/config.backup.$(date +%Y-%m-%d.%H:%M:%S)
                </Code>
                <Text>
                    merge your default config with the edgenet configuration:
                </Text>
                <Code>
                    KUBECONFIG=$HOME/.kube/config:$HOME/.kube/edgenet-{user.authority}-{user.name}.yml \ <br />
                    &nbsp;&nbsp;&nbsp;&nbsp; kubectl config view --merge --flatten > ~/.kube/config
                </Code>
                <Text>
                    When use kubectl you can now easily switch contexts to select the cluster you want to work on:
                </Text>
                <Code>
                    kubectl get pods --context=edgenet
                </Code>
                <Text>
                    If you don't want to merge the configuration files you can temporarily use the edgenet context in your shell
                    session like so:
                </Text>
                <Code>
                    export KUBECONFIG=$HOME/.kube/config:$HOME/.kube/edgenet-{user.authority}-{user.name}.yml
                    <br /><br />
                    kubectl get pods --context=edgenet
                </Code>

                <Text margin={{bottom:'medium'}}>
                    You will find more information about kubectl at the following links:
                    <ul>
                        <li>
                            <Anchor target="_blank" href="https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig/"
                                    label="Organizing Cluster Access Using kubeconfig Files" />
                        </li>
                        <li>

                            <Anchor target="_blank" href="https://kubernetes.io/docs/reference/kubectl/kubectl/" label="kubectl CLI" />
                        </li>
                        <li>
                            <Anchor target="_blank" href="https://kubernetes.io/docs/reference/kubectl/cheatsheet/" label="kubectl Cheat Sheet" />
                        </li>
                    </ul>

                </Text>
            </Box>

        </Box>
    );
}

export default Configuration;