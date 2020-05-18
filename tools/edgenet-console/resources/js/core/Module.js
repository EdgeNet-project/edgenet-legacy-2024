import React, {Suspense} from "react";

const Module = ({type, resource}) => {


    console.log(resource)

    const Component = React.lazy(() =>
        import('../modules/' + resource.api.type + '/' + type)
            .catch((err) => ({ default: () => {
                    console.log(err);
                    return <div>Not found</div>
                }
            }))
    );

    return (
        <Suspense fallback={<div>Loading...</div>}>
            <Component resource={resource} />
        </Suspense>
    )
}

export default Module;