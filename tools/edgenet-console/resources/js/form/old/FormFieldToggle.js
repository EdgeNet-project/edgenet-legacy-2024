import React from 'react';
import propTypes from "prop-types";
import { FormField, CheckBox } from "grommet";


const FormFieldToggle = (props) =>
    <FormField component={CheckBox} toggle {...props} plain={false} />;

export default FormFieldToggle;