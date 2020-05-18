import React from "react";
import { Box, FormField, TextInput, TextArea, Select, CheckBox } from "grommet";
import DateInput from "../form/ui/DateInput";
import ItemInput from "../form/input/ItemInput";

const NewsForm = () =>
    <Box>
        <FormField plain name="active" label="Activé">
            <CheckBox name="active" />
        </FormField>
        <FormField plain name="category" label="Categorie">
            <Select name="category" options={[
                'PROCESSUS', 'NOUS', 'GRAMMAIRE'
            ]} />
        </FormField>
        <DateInput name="date" label="Date" />
        <FormField name="text" label="Texte">
            <TextArea name="text" />
        </FormField>
        <FormField name="project_id" label="Projet" resource="projects" component={ItemInput}>
        </FormField>

    </Box>;

export default NewsForm;
