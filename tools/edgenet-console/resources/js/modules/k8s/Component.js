import React from "react";
import {K8s} from "./K8s";
import ListComponent from "./ui/List";

const Component = ({resource}) =>
    <K8s api={resource.api} currentId={null}>
        <ListComponent>
            {/*<ResourceList onClick={(id) => history.push('/' + '/' + id)} />*/}
        </ListComponent>
    </K8s>;

export default Component;