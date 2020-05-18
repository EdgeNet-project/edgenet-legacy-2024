import React from "react";
import { withRouter } from "react-router";
import axios from "axios";
import { Box } from "grommet";
import { Edit, Add } from "grommet-icons";
import { Button } from './ui';

class View extends React.Component {

    constructor(props) {
        super(props);
        this.state = {
            resource: null,
            id: null,
            item: null,
            isLoading: true
        };

        this.load = this.load.bind(this);
    }

    componentDidMount() {
        const { resource, id } = this.props.match.params;
        this.setState({
            resource: resource, id: id
        }, this.load)
    }

    componentDidUpdate(prevProps, prevState, snapshot) {
        const { resource, id } = this.state;

        if (resource !== prevState.resource || id !== prevState.id) {
            this.load();
        }
    }

    static getDerivedStateFromProps(props, state) {
        if (state.resource !== props.match.params.resource || state.id !== props.match.params.id) {
            return {
                resource: props.match.params.resource,
                id: props.match.params.id,
                item: null,
                isLoading: true
            };
        }

        return null;
    }

    load() {
        const { resource, id } = this.state;

        if (!resource || !id) {
            return;
        }

        axios.get('/api/' + resource + '/' + id)
            .then(({data}) => this.setState({
                item: this.sanitize(data),
                isLoading: false
            }))
            .catch((error) => console.log(error))

    }

    sanitize(data) {
        /**
         * see:
         * https://github.com/facebook/react/issues/11417
         * https://github.com/reactjs/rfcs/pull/53
         *
         */

        Object.keys(data).forEach((key, idx) => {
            if (data[key] === null) {
                data[key] = '';
            }
        });

        return data;
    }

    render() {
        const { children, label = '' } = this.props;
        const { item, resource, id, isLoading } = this.state;

        const view = item ?
            React.Children.map(children, (child, i) =>
                <Box key={child.key + i}>
                    <Box>
                        <Button icon={<Edit />} path={'/admin/' + resource + '/' + id + '/edit'} label={"Edit " + label} />
                    </Box>
                    { React.cloneElement(child, {
                        item: item, load: this.load
                    }) }
                </Box>) : <Box />;

        return (
            <Box>
                <Box pad={{vertical:'small'}} margin={{bottom:'small'}} border={{side:'bottom',size:'small',color:'brand'}}>
                    <Button icon={<Add />} path={'/admin/' + resource + '/new'} label={"New " + label} />
                </Box>
                {isLoading ? <Box pad="medium">...</Box> : view}
            </Box>
        );

    }
}

export default withRouter(View);
