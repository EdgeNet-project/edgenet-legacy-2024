import React from "react";
import { Image } from "grommet";
import { DataSourceConsumer } from "../DataSource";

// TOFIX
const FormFieldVideo = ({name}) =>
    <DataSourceConsumer>
        {
            ({ item }) => item[name] ? <Image src={item[name]} /> : null
        }
    </DataSourceConsumer>;

export default FormFieldVideo;