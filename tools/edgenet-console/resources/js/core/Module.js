import React, {Suspense} from "react";

const ModuleNotFount = ({err}) =>
    <div>Not found <br/>{err}</div>;

const ModuleLoading = () =>
    <div>Loading...</div>;

const Module = ({type, resource}) => {


    console.log(resource)

    const Component = React.lazy(() =>
        import('../modules/' + resource.api.type + '/' + type)
            .catch((err) => ({ default: () => {
                    return <ModuleNotFount error={err} />
                }
            }))
    );

    return (
        <Suspense fallback={<ModuleLoading />}>
            <Component resource={resource} />
        </Suspense>
    )
}

export default Module;