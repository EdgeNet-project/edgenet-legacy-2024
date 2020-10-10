import React from 'react';

import { TextInput } from "grommet";

const LoginInput = ({ value, onChange, placeholder = "E-Mail", ...rest }) => {
    return (
        <TextInput type="text" value={value} placeholder={placeholder}
                   onChange={event => onChange(event.target.value)}
                   {...rest}
        />
    );
};

export default LoginInput;
