import React, {useContext, useRef} from "react";
import {Box, Heading, Button, TextArea} from "grommet";
import {Copy, Dashboard} from "grommet-icons";
import { AuthenticationContext } from "../../authentication";

const Kubernetes = () => {
    const { user } = useContext(AuthenticationContext);
    const textareaEl = useRef(null);

    const copyToClipboard = () => {
        // textareaEl.current.focus();
        textareaEl.current.select()
        document.execCommand("copy")
    }

    return (
        <Box pad="medium">
            <Heading size="small" margin="none">
                Access Kubernetes Dashboard
            </Heading>

            <Box>
                You can access the EdgeNet Kubernetes dashboard by clicking on the button below and providing the following token to authenticate:
            </Box>

            <Box>
                <Box direction="row" gap="medium" margin={{vertical:'medium'}} justify="end">
                    <Button plain label="Copy to clipboard" icon={<Copy />} onClick={copyToClipboard} />
                </Box>
                <TextArea ref={textareaEl} rows="4" value={user.api_token} />
            </Box>
            <Box align="center" pad={{top:'large'}}>
                <Button primary target="_blank" href="https://dashboard.edge-net.org" label="Access Dashboard" icon={<Dashboard />} />
            </Box>

        </Box>
    );
}

export default Kubernetes;