import React, {useState, useContext, useEffect} from "react";
import axios from "axios";
import Select from "react-select";
import {RegistrationContext} from "../RegistrationContext";

const AuthoritySelect = () => {
    const { setAuthority } = useContext(RegistrationContext);
    const [ authorities, setAuthorities ] = useState([]);

    useEffect(() => {
        axios.get('/apis/apps.edgenet.io/v1alpha/authorities')
            .then(({data}) =>
                setAuthorities(data.items.map(item => {
                        return { value: item.metadata.name, label: item.spec.fullname + ' ('+item.spec.shortname+')' }
                }))
            )
    }, [authorities]);

    return (

        <Select placeholder="Select your institution"
                isSearchable={true} isClearable={true}
                options={authorities}
            // value={}
                name=""
                onChange={setAuthority}/>
    );
}

export default AuthoritySelect;