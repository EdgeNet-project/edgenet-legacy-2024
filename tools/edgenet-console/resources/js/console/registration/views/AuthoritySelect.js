import React, {useState, useContext, useEffect} from "react";
import axios from "axios";
import Select from "react-select";
import { RegistrationContext } from "../RegistrationContext";

const AuthoritySelect = () => {
    const { setAuthority } = useContext(RegistrationContext);
    const [ authorities, setAuthorities ] = useState([]);

    useEffect(() => {
        axios.get('/apis/apps.edgenet.io/v1alpha/authorities')
            .then(({data}) =>
                data.items && setAuthorities(data.items.map(item => {
                        return {
                            value: item.metadata.name,
                            label: item.spec.fullname + ' ('+item.spec.shortname+')'
                        }
                }))
            )
    }, []);

    const selectAuthority = (selected) => {
        if (selected && selected.value) {
            setAuthority(selected.value)
        } else {
            setAuthority(null)
        }

    }

    return <Select placeholder="Select your institution"
                   isSearchable={true} isClearable={true}
                   options={authorities}
                   name="authority"
                   onChange={selectAuthority}/>;
}

export default AuthoritySelect;