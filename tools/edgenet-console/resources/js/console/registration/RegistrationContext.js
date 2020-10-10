import React, {useState} from "react";
import axios from "axios";

const RegistrationContext = React.createContext({});
const RegistrationConsumer = RegistrationContext.Consumer;

const Registration = ({children}) => {
    const [ authority, setAuthority ] = useState(null);
    const [ message, setMessage ] = useState(null);
    const [ errors, setErrors ] = useState({});
    const [ loading, setLoading ] = useState(null);
    const [ success, setSuccess ] = useState(false);

    const submitRegistration = ({value}) => {

        if (!authority) {
            setMessage('Authority not selected')
            return;
        }

        setLoading(true)

        axios.post('/register', {
            ...value,
            authority: authority
        })
            .then(({data}) => {
                console.log(data)

                setSuccess(true)
                setMessage(null)
                setErrors({})
            })
            .catch(error => {
                setMessage(error.message)

                if (error.response) {
                    console.log(1, error.response.data);
                    setMessage(error.response.data.message)
                    if (error.response.data.errors) {
                        setErrors(error.response.data.errors)
                    }
                } else if (error.request) {
                    console.log(2, error.request);
                } else {
                    console.log('client error');
                    console.log(3, error);
                }
            })
            .finally(() => setLoading(false))

    }

    return (
        <RegistrationContext.Provider value={{
            setAuthority: setAuthority,
            submitRegistration: submitRegistration,
            loading: loading,
            message: message,
            errors: errors,
            success: success
        }}>
            {children}
        </RegistrationContext.Provider>

    )

}

export {Registration, RegistrationContext, RegistrationConsumer };