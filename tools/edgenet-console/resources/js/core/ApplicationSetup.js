import React from "react";

const ApplicationContext = React.createContext({
    resources: []
});
const ApplicationConsumer = ApplicationContext.Consumer;

class ApplicationSetup extends React.Component {

    constructor(props) {
        super(props);

        this.state = {}
    }

    componentDidMount() {
    }

    render() {
        const { children, menu, resources } = this.props;

        return (
            <ApplicationContext.Provider value={{
                menu: menu,
                resources: resources,
            }}>
                {children}
            </ApplicationContext.Provider>
        );
    }
}


export {
    ApplicationSetup, ApplicationContext, ApplicationConsumer
}
