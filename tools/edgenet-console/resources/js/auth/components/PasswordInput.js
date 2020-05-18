import React, { useState } from 'react';

import { Stack, TextInput, Button } from "grommet";
import { FormLock, View } from "grommet-icons";

const PasswordInput = ({ value, onChange, placeholder = "Password", ...rest }) => {
    const [reveal, setReveal] = useState(false);

    return (
        <Stack anchor="right">
            <TextInput type={reveal ? "text" : "password"} value={value}
                       onChange={event => onChange(event.target.value)} placeholder={placeholder}
                       {...rest}
            />
            <Button icon={reveal ? <FormLock size="medium" /> : <View size="medium" />}
                    onClick={() => setReveal(!reveal)}
            />
        </Stack>
    );
};

export default PasswordInput;
