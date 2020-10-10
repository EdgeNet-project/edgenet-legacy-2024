import React from 'react';
import propTypes from "prop-types";
import { FormField } from "grommet";

const FormFieldTextArea = (props) =>
    <FormField {...props} plain={false} />;

export default FormFieldTextArea;