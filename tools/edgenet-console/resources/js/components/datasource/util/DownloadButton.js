import React from "react";
import { Button } from "grommet";
import { Download, DocumentExcel } from "grommet-icons";
import LocalizedStrings from "react-localization";

import { DataSourceConsumer } from "../DataSource";

const strings = new LocalizedStrings({
    en: {
        exportdoc: "Export {0} {1}",
    },
    fr: {
        exportdoc: "Exporter {0} {1}",
    }
});

const DownloadButton = ({type, label, ...props}) =>
    <DataSourceConsumer>
        {
            ({downloadItems, itemsDownloading, count}) =>
                <Button disabled={itemsDownloading} icon={itemsDownloading ? <Download /> : <DocumentExcel />}
                        label={strings.formatString(strings.exportdoc, count, label)}
                        onClick={() => downloadItems(type)}
                    {...props}
                />
        }
    </DataSourceConsumer>;


export default DownloadButton;