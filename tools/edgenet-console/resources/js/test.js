import React from "react";
import { Box, Form, FormField, Button} from "grommet";

const Test = () =>
    <Form onSubmit={({ value }) => {}}>
        <FormField name="name" htmlfor="textinput-id" label="Name" />
        <Box direction="row" gap="medium">
            <Button type="submit" primary label="Submit" />
            <Button type="reset" label="Reset" />
        </Box>
    </Form>

export default Test;
