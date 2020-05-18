import React from "react";
import LocalizedStrings from "react-localization";
import { Box, Form as GrommetForm, Button } from "grommet";
import { Save } from "grommet-icons";
import { FormConsumer } from "./Form";

const strings = new LocalizedStrings({
    en: {
        reset: "Reset",
        save: "Save"
    },
    fr: {
        reset: "Réinitialiser",
        save: "Sauvegarder"
    }
});

const FormXX = ({children, onSubmit}) =>
    <FormConsumer>
        {
            ({item, save, load, changed, setChanged}) =>
                <GrommetForm value={item || {}}
                             onReset={load}
                             onChange={() => setChanged(true)}
                             onSubmit={({ value }) => save(value)}>
                    <Box>
                        {children}
                    </Box>
                    <Box direction="row" justify="start" pad={{vertical:'xsmall'}}>
                        <Box pad="small">
                            <Button plain icon={<Save />} disabled={!changed} type="submit" label={strings.save} />
                        </Box>
                    </Box>
                </GrommetForm>
        }
    </FormConsumer>;

export default FormXX;
