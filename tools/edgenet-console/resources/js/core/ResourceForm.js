import React, {Suspense} from "react";
import {Form} from "../form";

const ResourceForm = ({match}) => {
    const ResourceForm = React.lazy(() =>
        import('../views/' + match.params.resource.charAt(0).toUpperCase() + match.params.resource.slice(1) + 'Form')
            .catch((err) => ({ default: () => {
                    console.log(err);
                    return <div>Not found</div>
                }
            }))
    );

    return (
        <Suspense fallback={<div>Loading...</div>}>
            <Form>
                <ResourceForm />
            </Form>
        </Suspense>
    )
};

export default ResourceForm;