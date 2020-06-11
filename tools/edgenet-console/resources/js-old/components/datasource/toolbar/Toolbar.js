import React from "react";

import { Box, Stack, Text, Button } from "grommet";
import { Close } from "grommet-icons";



const ToolbarIcon = ({icon, count}) =>
    <div style={{position:'relative'}}>
        {icon}
        {count > 0 && <Box background="brand" pad={{ horizontal: 'xsmall' }}
                           style={{position:'absolute',top:-8,right:-6}} round>
            <Text size="xsmall">{count}</Text>
        </Box>}
    </div>;

class ToolbarTab extends React.Component {

    constructor(props) {
        super(props);
        this.state = { hover: false }
    }

    render() {
        let { label='Tab', icon=null, count=0, current, name, onClick } = this.props;
        let { hover } = this.state;

        let round = hover && current === null ? 'xsmall' : {size: "xsmall", corner: "top"};
        let background = current === name ? "light-4" : hover ? "light-2" : null;

        return (
            <Box pad="small" round={round}
                 background={background}
                 onMouseEnter={() => this.setState({hover: true})}
                 onMouseLeave={() => this.setState({hover: false})}
                 onClick={onClick}
            >
                <Button label={label} plain icon={<ToolbarIcon icon={icon} count={count} />} />
            </Box>
        )
    }
}

class ToolbarButton extends React.Component {

    constructor(props) {
        super(props);
        this.state = { hover: false }
    }

    render() {
        let { hover } = this.state;
        let { label, icon, active, onClick } = this.props;

        return (
            <Box background={active ? "brand" : (hover ? "light-3" : null)} round="small" pad="xsmall"
                 onMouseEnter={() => this.setState({hover: true})}
                 onMouseLeave={() => this.setState({hover: false})}
                 onClick={onClick}
            >
                <Button icon={icon} label={label} plain />
            </Box>
        );
    }
}

class Toolbar extends React.PureComponent {

    constructor(props) {
        super(props);
        this.state = {
            showTool: null,
            error: ''
        };
        this.handleToolbarTab = this.handleToolbarTab.bind(this);
    }

    componentDidMount() {

    }

    componentDidCatch(error, errorInfo) {
        this.setState({error: error});
    }

    handleToolbarTab(name) {
        this.setState({
            showTool: (this.state.showTool === name) ? null : name
        })
    }


    render() {
        let { children } = this.props;
        let { showTool, error } = this.state;

        if (error) {
            return error;
        }

        let currentTool = null;
        let toolbar = React.Children.map(children, (child) => {
            let { Tab, Button, name } = child.type;

            if (showTool === name) {
                currentTool = <Box background="light-4">{child}</Box>;
            }

            if (Tab !== undefined) {
                return <Tab name={name} current={showTool}
                            onClick={() => this.handleToolbarTab(name)}
                            {...child.props} />
            }

            if (Button !== undefined) {
                return <Button />
            }

            return child;

        });

        return (
            <Box flex={false} margin={{vertical:'xsmall'}}>
                <Box direction="row">
                    {toolbar}
                </Box>
                {currentTool}
            </Box>
        );
    }
}

export { Toolbar, ToolbarTab, ToolbarButton };
