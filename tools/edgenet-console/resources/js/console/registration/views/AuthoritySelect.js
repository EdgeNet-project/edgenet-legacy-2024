import React, {useState, useEffect} from "react";
import axios from "axios";
import Select from "react-select";

const AuthoritySelect = ({setAuthority}) => {
    const [ authorities, setAuthorities ] = useState([]);

    useEffect(() => {
        axios.get('/apis/apps.edgenet.io/v1alpha/authorities')
            .then(({data}) => {
                if (data.items) {
                    data.items.sort(compare)
                    setAuthorities(data.items.map(item => {
                        return {
                            value: item.metadata.name,
                            label: item.spec.fullname + ' (' + item.spec.shortname + ')'
                        }
                    }));
                }
            })
    }, []);

    const compare = ( a, b ) => {
        if ( a.spec.fullname  < b.spec.fullname) {
            return -1;
        }
        if ( a.spec.fullname  > b.spec.fullname) {
            return 1;
        }
        return 0;
    }


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