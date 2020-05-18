import React, {Suspense} from "react";
import {View} from "../view";

const ResourceView = ({match}) => {
    const ResourceView = React.lazy(() =>
        import('../views/' + match.params.resource.charAt(0).toUpperCase() + match.params.resource.slice(1) + 'View')
            .catch(() => ({ default: () => <div>Not found</div> }))
    );

    return (
        <Suspense fallback={<div>Loading...</div>}>
            <View>
                <ResourceView />
            </View>
        </Suspense>
    )
};

export default ResourceView;