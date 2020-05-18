import React, {Suspense} from "react";
//
//
//
//
// import {BrowserRouter as Router, Route, Switch} from "react-router-dom";
// import {Grommet} from "grommet";
//
// import {AuthProvider} from "../auth";
// import {Authenticated, Guest} from "../auth/access";
// import {ForgotPasswordView, LoginView, ResetPasswordView} from "../auth/views";
//
// import {NavigationView} from "../nav/views";
// import {Form, Related} from "../form";
//
// import ResourceList from "./ResourceList";
// import {View} from "../view";
//
// const ResourceList = ({match, history}) => {
//     const ResourceList = React.lazy(() =>
//         import('../views/' + match.params.resource.charAt(0).toUpperCase() + match.params.resource.slice(1) + 'List')
//             .catch(() => ({ default: () => <div>Not found</div> }))
//     );
//
//     return (
//         <Suspense fallback={<div>Loading...</div>}>
//             <K8s resource={"/api/" + match.params.resource} currentId={match.params.id}>
//                 <List>
//                     <ResourceList onClick={(id) => history.push('/' + match.params.resource + '/' + id)} />
//                 </List>
//             </K8s>
//         </Suspense>
//     )
//
// };



const ConsoleContext = React.createContext({
    resources: []
});
const ConsoleConsumer = ConsoleContext.Consumer;


class Console extends React.Component {

    constructor(props) {
        super(props);

        this.state = {}

        // this.loadComponent = this.loadComponent.bind(this);
    }

    componentDidMount() {
    }
    //
    // loadModule(resourceName, componentType) {
    //     const { resources } = this.props;
    //
    //     const resource = resources.find(r => r.name === resourceName);
    //
    //     if (!resource) {
    //         throw 'Resource ' + resourceName + ' not configured';
    //     }
    //
    //     console.log(resource)
    //
    //     const Component = React.lazy(() =>
    //         import('../modules/' + resource.api.type + '/' + componentType)
    //             .catch((err) => ({ default: () => {
    //                     console.log(err);
    //                     return <div>Not found</div>
    //                 }
    //             }))
    //     );
    //
    //     return (
    //         <Suspense fallback={<div>Loading...</div>}>
    //             <Component resource={resource} />
    //         </Suspense>
    //     )
    // }

    render() {
        const { children, resources } = this.props;

        return (
            <ConsoleContext.Provider value={{
                resources: resources,
                // loadComponent: this.loadComponent
            }}>
                {children}
            </ConsoleContext.Provider>
        );
    }
}


export { Console, ConsoleContext, ConsoleConsumer };
