import React, {Suspense} from "react";
import {K8s} from "../K8s";
import List from "../ui/List";

const Component = ({resource}) => {
    const ListComponent = React.lazy(() =>
        import('../../../resources/' + resource.name + '/List')
            .catch((err) => ({ default: () => {
                    console.log(err);
                    return <div>Not found</div>
                }
            }))
    );
    console.log('resource loading')

    return (
    <K8s api={resource.api} currentId={null}>
        <List>
            {/*<Suspense fallback={<div>Resource Loading...</div>}>*/}
                <ListComponent resource={resource} />
            {/*</Suspense>*/}
            {/*<ResourceList onClick={(id) => history.push('/' + '/' + id)} />*/}
        </List>
    </K8s>
    );
};

export default Component;