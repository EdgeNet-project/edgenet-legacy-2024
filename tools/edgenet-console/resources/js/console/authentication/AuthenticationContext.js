import React, {useState, useEffect} from 'react';
import axios from 'axios';

const AuthenticationContext = React.createContext({});
const AuthenticationConsumer = AuthenticationContext.Consumer;

const Authentication = ({children}) => {
    const [ token, setToken ] = useState(sessionStorage.getItem('api_token', null));
    const [ user, setUser ] = useState({});
    const [ aup, setAup ] = useState(null);
    const [ edgenet, setEdgenet ] = useState(null);
    const [ message, setMessage ] = useState(null);
    const [ loading, setLoading ] = useState(true);

    useEffect(() => {
        (token) ? getUser() : setLoading(false);
    }, [token]);

    useEffect(() => {
            if (user && user.api_token) {
                axios.defaults.headers.common = {
                    Authorization: "Bearer " + user.api_token
                };
                sessionStorage.setItem('api_token', user.api_token);

                axios.all([getAUP(), getEdgenet()]).finally(() => setLoading(false))
            }
        },
        [user]);

    // axios.defaults.headers.common = {
    //     'Accept': 'application/json',
    //     'Content-Type': 'application/json'
    // };



    const getUser = () => {
        return axios.get('/api/user', { headers: { Authorization: "Bearer " + token } })
            .then(({data}) => {
                if (!data.api_token) {
                    error('invalid token');
                } else {
                    setUser(data);
                }

            })
            .catch(error => {
                if (error.response) {
                    // The request was made and the server responded with a status code
                    // that falls out of the range of 2xx
                    console.log(error.response.data);
                    console.log(error.response.status);
                    console.log(error.response.headers);
                } else if (error.request) {
                    // The request was made but no response was received
                    // `error.request` is an instance of XMLHttpRequest in the browser and an instance of
                    // http.ClientRequest in node.js
                    console.log(error.request);
                } else {
                    // Something happened in setting up the request that triggered an Error
                    console.log('Error', error.message);
                }
                console.log(error.config);

                setUser({});
                sessionStorage.removeItem('api_token')
            });
    }

    const getAUP = () => {
        return axios.get('/apis/apps.edgenet.io/v1alpha/namespaces/authority-' + user.authority + '/acceptableusepolicies/' + user.name)
            .then(({data}) => setAup(data.spec))
            .catch(error => console.log(error))
    }

    const getEdgenet = () => {
        return axios.get('/apis/apps.edgenet.io/v1alpha/namespaces/authority-' + user.authority + '/users/' + user.name)
            .then(({data}) => setEdgenet(data))
            .catch(err => console.log(err));
    }

    const setError = (message) => {
        setMessage(message)
        setLoading(false)

        setTimeout(() => setMessage(null), 6000)
    }

    const login = (email, password) => {
        setLoading(true)
        axios.post('/login', {
            email: email,
            password: password,
        })
        .then(({data}) => setUser(data))
        .catch((error) => {
            if (error.response) {
                setError(error.response.data.message || '');
            } else if (error.request) {
                setError('server is not responding, try later');
            } else {
                setError('client error');
            }
        })
        .finally(() => setLoading(false))

    }

    const logout = () => {
        setLoading(true)
        axios.post('/logout')
        .then((response) => {
            setToken(null);
            setUser({});
            setEdgenet(null);
            setAup(null);
            sessionStorage.removeItem('api_token');
        })
        .catch((error) => {
            if (error.response) {
                setError(error.response.data.message || '');
            } else if (error.request) {
                setError('server is not responding, try later');
            } else {
                setError('client error');
            }
        })
        .finally(() => setLoading(false));
    }

    const isAuthenticated = () => {
        return !!user.api_token;
    }

    const isGuest = () => {
        return !isAuthenticated();
    }

    const isAdmin = () => {
        return isAuthenticated() && edgenet.status && edgenet.status.type === 'admin';
    }

    const isClusterAdmin = () => {
        return isAuthenticated() && user.admin;
    }

    const sendResetLink = (email) => {
        setLoading(true)
        axios.post('/password/email', {
            email: email,
        })
        .then(({data}) => setMessage("an email will be sent to you"))
        .catch((error) => {
            if (error.response) {
                setError(error.response.data.message || '');
            } else if (error.request) {
                setError('server is not responding, try later');
            } else {
                setError('client error');
            }
        })
        .finally(() => setLoading(false));
    }

    const resetPassword = (email, token, password, password_confirmation) => {
        axios.post('/password/reset', {
            email: email,
            token: token,
            password: password,
            password_confirmation: password_confirmation
        })
        .then(({data}) => setMessage("password updated succesfully"))
        .catch((error) => {
            if (error.response) {
                setError(error.response.data.message || '');
            } else if (error.request) {
                setError('server is not responding, try later');
            } else {
                setError('client error');
            }
        })
        .finally(() => setLoading(false))
    }



    if (token && !isAuthenticated()) {
        // checking if token is valid
        return null;
    }

    return (
        <AuthenticationContext.Provider value={{
            user: user,
            aup: aup,
            edgenet: edgenet,


            login: login,
            logout: logout,

            isAuthenticated: isAuthenticated,
            isGuest: isGuest,
            isAdmin: isAdmin,
            isClusterAdmin: isClusterAdmin,

            sendResetLink: sendResetLink,
            resetPassword: resetPassword,

            loading: loading,
            message: message,

            getAUP: getAUP

        }}>
            {children}
        </AuthenticationContext.Provider>
    );

}

export { Authentication, AuthenticationContext, AuthenticationConsumer };
