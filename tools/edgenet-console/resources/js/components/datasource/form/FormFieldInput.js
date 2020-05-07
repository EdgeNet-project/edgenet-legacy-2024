import React from 'react';
import propTypes from "prop-types";
import { FormField } from "grommet";

const FormFieldInput = (props) =>
    <FormField {...props} plain={false} />;

export default FormFieldInput;