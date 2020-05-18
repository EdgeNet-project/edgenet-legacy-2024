import React from "react";
import { Box, FormField, TextInput, TextArea, CheckBox } from "grommet";

const GrammaireForm = () =>
    <Box>
        <FormField plain name="active" label="Activé">
            <CheckBox name="active" />
        </FormField>
        <FormField name="title" label="Titre">
            <TextInput name="title" />
        </FormField>
        <FormField name="text" label="Texte">
            <TextArea name="text" />
        </FormField>

    </Box>;

export default GrammaireForm;
