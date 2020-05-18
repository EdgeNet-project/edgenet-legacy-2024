import React, { Suspense } from "react";
import { Box, Tabs, Tab } from "grommet";
import { ApplicationContext, ApplicationConsumer } from "../core/ApplicationSetup";


const ItemsTab = (props) => {
    const Component = React.lazy(() => import('./tabs/Items')
        .catch(() => ({ default: () => <div>Not found</div> }))
    );

    return (
        <Suspense fallback={<div>Loading...</div>}>
            <Component {...props} />
        </Suspense>
    )

};

const MediaTab = ({component, ...props}) => {
    const Component = React.lazy(() => import('./tabs/' + component.charAt(0).toUpperCase() + component.slice(1))
            .catch(() => ({ default: () => <div>Not found</div> }))
    );

    return (
        <Suspense fallback={<div>Loading...</div>}>
            <Component {...props} />
        </Suspense>
    )

};

class Related extends React.Component {

    constructor(props) {
        super(props);
        this.state = {
            index: 0,
            media: [],
            related: []
        }
    }

    componentDidMount() {
        const { resource } = this.props.match.params;
        const { resources } = this.context;

        const r = resources.find(r => r.name === resource);

        this.setState({
            media: (Array.isArray(r.media) && r.media.length > 0) ? r.media : [],
            related: (Array.isArray(r.related) && r.related.length > 0) ? r.related : []
        });
    }



    render() {
        const { resource, id } = this.props.match.params;
        const { index, media, related } = this.state;

        if (!media.length && !related.length) return null;

        return (
            <Tabs activeIndex={index} onActive={(index) => this.setState({index: index})}>
                {media.map(m =>
                    <Tab key={m} title={MediaTab.title || m}>
                        <Box pad="medium">
                            <MediaTab component={m} resource={resource} id={id} />
                        </Box>
                    </Tab>
                )}

                {related.map(m =>
                    <Tab key={m} title={ItemsTab.title || m}>
                        <Box pad="medium">
                            <ItemsTab related={m} resource={resource} id={id} />
                        </Box>
                    </Tab>
                )}
            </Tabs>
        );
    }
}

Related.contextType = ApplicationContext;

export default Related;
