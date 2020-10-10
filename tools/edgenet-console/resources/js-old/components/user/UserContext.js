import React from 'react';
import axios from 'axios';

const UserContext = React.createContext({});
const UserConsumer = UserContext.Consumer;

class UserProvider extends React.Component {

    constructor(props) {
        super(props);
        this.state = {
            user: {},
            message: '',
            loading: true
        };

        // this.csrf = null;
        // try {
        //     this.csrf = document.querySelector("meta[name='csrf-token']").getAttribute('content')
        // } catch (e) {
        //     console.log(e.message);
        // }

        axios.defaults.headers.common = {
            // 'Access-Control-Allow-Origin': "http://localhost1234",
            'Accept': 'application/json',
            'Content-Type': 'application/json'
        };
        // axios.defaults.withCredentials = true;
        // axios.defaults.credentials = 'same-origin';
        // axios.defaults.mode = 'no-cors';
        // axios.defaults.baseURL = 'https://edgenet.planet-lab.eu:6443/apis/apps.edgenet.io/v1alpha';

        this.token = sessionStorage.getItem('api_token');

        this.setUser = this.setUser.bind(this);
        this.login = this.login.bind(this);
        this.logout = this.logout.bind(this);
        this.sendResetLink = this.sendResetLink.bind(this);
    }

    componentDidMount() {
        if (this.token) {
            const { auth } = this.props;
            axios.get(auth, { headers: { Authorization: "Bearer " + this.token } })
                .then(({data}) => this.setUser(data))
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
                    this.setState({
                        user: {}, loading: false,
                    }, () => sessionStorage.removeItem('api_token'))
                });
        } else {
            this.setState({loading: false}, this.setAnonymous);
        }
    }

    componentWillUnmount() {
    }

    setUser(user) {
        if (!user.token) {
            this.error('invalid token');
            return false;
        }

        axios.defaults.headers.common = {
            Authorization: "Bearer " + user.token
        };

        this.setState({
            user: user,
            loading: false
        }, () => {
            this.token = null;
            sessionStorage.setItem('api_token', user.token);
        });
    }

    error(message) {
        this.setState({ message: message, loading: false })
    }

    login(email, password) {

        this.setState({ loading: true }, () =>
            axios.post('/login', {
                email: email,
                password: password,
                //csfr: this.csrf
            })
                .then(({data}) => this.setUser(data))
                .catch((error) => {
                    if (error.response) {
                        this.error(error.response.data.message || '');
                    } else if (error.request) {
                        this.error('server is not responding, try later');
                    } else {
                        this.error('client error');
                    }
                })
        );
    }

    logout() {
        const { logout } = this.props;
        axios.post(logout)
            .then((response) => {
                this.setState({
                    user: {},
                }, () => sessionStorage.removeItem('api_token'))
            })
            .catch((error) => {
                if (error.response) {
                    this.error(error.response.data.message || '');
                } else if (error.request) {
                    this.error('server is not responding, try later');
                } else {
                    this.error('client error');
                }
            });
    }

    sendResetLink(email) {
        let { reset } = this.props;

        // const { resetLink } = this.props;
        axios.post(reset, {
            email: email
        })
            .then((response) => {
                this.setState({
                    message: "lien envoyé",
                })
            })
            .catch((error) => {
                if (error.response) {
                    this.error(error.response.data.message || '');
                } else if (error.request) {
                    this.error('server is not responding, try later');
                } else {
                    this.error('client error');
                }
            });
    }

    render() {
        let { children, passwordResetLink, loginPath } = this.props;
        let { user, message, loading } = this.state;

        if (this.token && !user.token) {
            // still loading
            return null;
        }

        return (
            <UserContext.Provider value={{
                user: user,
                login: this.login,
                logout: this.logout,
                sendResetLink: this.sendResetLink,
                loading: loading,
                message: message,

                passwordResetLink: passwordResetLink,
                loginPath: loginPath
            }}>{children}</UserContext.Provider>
        );
    }
}

UserProvider.propTypes = {
};

UserProvider.defaultProps = {
};

export { UserContext, UserProvider, UserConsumer };
