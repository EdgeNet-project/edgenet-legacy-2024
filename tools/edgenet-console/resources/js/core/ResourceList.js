import React from "react";
import { ConsoleConsumer } from "./Console";
import Module from "./Module";

class ResourceList extends React.Component {

    constructor(props) {
        super(props);

    }

    componentDidMount() {
        const { resource, id } = this.props.match.params;

    }

    render() {
        const { resource, id } = this.props.match.params;

        if (!resource) {
            throw 'Resource ' + resourceName + ' not configured';
        }

        return (
            <ConsoleConsumer>
                {({resources}) => <Module type="list" resource={resources.find(r => r.name === resource)} />}
            </ConsoleConsumer>
        )

        // return (
        //     <ConsoleConsumer>
        //         {({loadComponent}) => loadComponent(resource, 'list')}
        //     </ConsoleConsumer>
        // )
    }
}


export default ResourceList;