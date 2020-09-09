import React, {Suspense, useContext, useState} from "react";
import { useParams } from "react-router-dom"
import axios from "axios";

import { ConsoleContext } from "../../index";
import { Box, InfiniteScroll } from "grommet";

import {
    NodeList,

    NotFound
} from "../components";

//
// const ListComponent = ({r}) => {
//
//     return React.lazy(() =>
//         import('../../../resources/' + r.name + '/List')
//             .catch((err) => ({ default: () => {
//                     console.log(err);
//                     return <NotFound />
//                 }
//             }))
//     );
// }

const ListComponent = () => {
    const { resource, id } = useParams();

    switch(resource) {
        case 'nodes':
            return <NodeList resource={resource} />
        default:
            return <NotFound />
    }
}

const ListRow = ({children, item}) => {
    // const [ isMouseOver, set]


    // const { item, isActive, orderable, onClick } = this.props;
    // const { isMouseOver } = this.state;

    // const background = isActive ? 'light-4' : isMouseOver ? 'light-2' : 'light-1';

    children = React.cloneElement(children, {
        item: item,
        // isActive: isActive,
        // isMouseOver: isMouseOver,
    }, null);

    // if (orderable) {
    //     children = <OrderableItem isMouseOver={isMouseOver}
    //                               item={item}>{children}</OrderableItem>
    // }

    return (
        <Box
            // onMouseEnter={() => this.setState({ isMouseOver: true })}
            //  onMouseLeave={() => this.setState({ isMouseOver: false })}
            //  onClick={() => onClick(item)}
            //  background={background}
            border={{side:'bottom', color:'light-4'}}
            flex={false}>
            {children}
        </Box>
    );

}

const Nodes = () => {
    const [ resources, setResources ] = useState([]);
    const [ loading, setLoading ] = useState(false);
    const { config } = useContext(ConsoleContext);

    const getResources = () => {
        // const { items, current_page, last_page, queryParams } = this.state;

        // if (!api) return false;

        // if (current_page >= last_page) return;

        //const url = '';
        axios.get('/api/v1/nodes', {
            // params: { ...queryParams, page: current_page + 1 },
            // paramsSerializer: qs.stringify,
        })
            .then(({data}) => {
                setResources(data.items);
                // this.setState({
                //     ...data, loading: false
                // });
            })
            .catch(error => {
                console.log(error)
            });
    }

    if (loading) {
        return <Box>Loading</Box>;
    }

    return (
            <Box overflow="auto">
                <InfiniteScroll items={resources} onMore={getResources}
                    //step={per_page}
                    // show={currentIdx}
                    // renderMarker={marker => itemsLoading && <Box pad="medium" background="accent-1">{marker}</Box>}
                >
                    {(item, j) =>
                        <ListRow key={'items-' + j} item={item}>
                            <ListComponent resource={r} />
                        </ListRow>
                    }
                </InfiniteScroll>
            </Box>

        // <Module type="list" resource={resources.find(r => r.name === resource)} />
    )


}


export default Nodes;