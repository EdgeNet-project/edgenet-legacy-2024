import React from "react";
import { Button } from "grommet";
import { Download, DocumentExcel } from "grommet-icons";
import LocalizedStrings from "react-localization";

import { DataConsumer } from "../Data";

const strings = new LocalizedStrings({
    en: {
        exportdoc: "Export {0} {1}",
    },
    fr: {
        exportdoc: "Exporter {0} {1}",
    }
});

const DownloadButton = ({type, label, ...props}) =>
    <DataConsumer>
        {
            ({downloadItems, itemsDownloading, count}) =>
                <Button disabled={itemsDownloading} icon={itemsDownloading ? <Download /> : <DocumentExcel />}
                        label={strings.formatString(strings.exportdoc, count, label)}
                        onClick={() => downloadItems(type)}
                    {...props}
                />
        }
    </DataConsumer>;


export default DownloadButton;
