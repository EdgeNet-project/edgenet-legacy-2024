import React from "react";
import { Box, FormField, TextInput, TextArea, Select, CheckBox } from "grommet";

const ProjectsForm = () =>
    <Box>
        <FormField plain name="active" label="Activé">
            <CheckBox name="active" />
        </FormField>
        <FormField plain name="category" label="Categorie">
            <Select name="category" options={[
                'VILLE DENSE', 'VILLAGES ET BOURGS', 'GRAND SITES'
            ]} />
        </FormField>
        <FormField name="name" label="Nom">
            <TextInput name="name" />
        </FormField>
        <FormField name="description" label="Description">
            <TextArea name="description" />
        </FormField>
        <FormField name="details" label="Détails">
            <TextArea name="details" />
        </FormField>
    </Box>;

export default ProjectsForm;
