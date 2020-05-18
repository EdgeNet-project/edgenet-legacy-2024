import React from 'react';
import axios from 'axios';

const AuthContext = React.createContext({});
const AuthConsumer = AuthContext.Consumer;

class AuthProvider extends React.Component {

    constructor(props) {
        super(props);
        this.state = {
            user: {},
            message: '',
            loading: true
        };

        axios.defaults.headers.common = {
            'Accept': 'application/json',
            'Content-Type': 'application/json'
        };

        this.token = sessionStorage.getItem('api_token');

        this.getUser = this.getUser.bind(this);
        this.setUser = this.setUser.bind(this);
        this.login = this.login.bind(this);
        this.logout = this.logout.bind(this);
        this.isAuthenticated = this.isAuthenticated.bind(this);
        this.isGuest = this.isGuest.bind(this);
        this.sendResetLink = this.sendResetLink.bind(this);
        this.resetPassword = this.resetPassword.bind(this);

    }

    componentDidMount() {
        (this.token) ? this.getUser() : this.setState({loading: false});
    }

    componentWillUnmount() {
    }

    getUser() {
        axios.get('/api/user', { headers: { Authorization: "Bearer " + this.token } })
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
    }

    setUser(user) {
        if (!user.api_token) {
            this.error('invalid token');
            return false;
        }

        axios.defaults.headers.common = {
            Authorization: "Bearer " + user.api_token
        };

        this.setState({
            user: user,
            loading: false
        }, () => {
            this.token = null;
            sessionStorage.setItem('api_token', user.api_token);
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

    isAuthenticated() {
        const { user } = this.state;
        return !!user.api_token;
    }

    isGuest() {
        return !this.isAuthenticated();
    }

    sendResetLink(email) {
        this.setState({ loading: true }, () =>
            axios.post('/password/email', {
                email: email,
            })
                .then(({data}) => this.setState({
                    loading: false,
                    message: "an email will be sent to you"
                }))
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

    resetPassword(email, token, password, password_confirmation) {
        this.setState({ loading: true }, () =>
            axios.post('/password/reset', {
                email: email,
                token: token,
                password: password,
                password_confirmation: password_confirmation
            })
                .then(({data}) => this.setState({
                    loading: false,
                    message: "password updated succesfully"
                }))
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

    render() {
        let { children } = this.props;
        let { user, message, loading } = this.state;

        if (this.token && !this.isAuthenticated()) {
            // checking if token is valid
            return null;
        }

        return (
            <AuthContext.Provider value={{
                user: user,
                login: this.login,
                logout: this.logout,

                isAuthenticated: this.isAuthenticated,
                isGuest: this.isGuest,

                sendResetLink: this.sendResetLink,
                resetPassword: this.resetPassword,

                loading: loading,
                message: message,
            }}>
                {children}
            </AuthContext.Provider>
    );
    }
}

AuthProvider.propTypes = {
};

AuthProvider.defaultProps = {
};

export { AuthContext, AuthProvider, AuthConsumer };
