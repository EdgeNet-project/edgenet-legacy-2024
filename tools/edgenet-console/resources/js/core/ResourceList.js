import React, { Suspense, useContext } from "react";
import { useParams } from "react-router-dom"
import { ConsoleContext } from "./Console";
import { EdgenetContext } from "../edgenet";
import {Box, InfiniteScroll} from "grommet";

const NotFound = ({err}) =>
    <div>Not found <br/>{err}</div>;

const Loading = () =>
    <div>Loading...</div>;

const ListComponent = ({r}) => {

    return React.lazy(() =>
        import('../../../resources/' + r.name + '/List')
            .catch((err) => ({ default: () => {
                    console.log(err);
                    return <NotFound />
                }
            }))
    );
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

const ResourceList = () => {
    const { config } = useContext(ConsoleContext);
    const { resources, getResources } = useContext(EdgenetContext);
    const { resource, id } = useParams();

    const r = config.resources.find(r => r.name === resource)

    const onMore = () => {
        getResources(r.api)
    }

    return (
        <Suspense fallback={<Loading />}>
            <Box overflow="auto">
                <InfiniteScroll items={resources} onMore={onMore}
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
        </Suspense>

        // <Module type="list" resource={resources.find(r => r.name === resource)} />
    )


}


export default ResourceList;