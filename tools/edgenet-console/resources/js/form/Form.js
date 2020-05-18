import React from "react";
import { withRouter } from "react-router";
import axios from "axios";
import PropTypes from "prop-types";
import {Box, Form as GrommetForm, Button} from "grommet";
import {Save} from "grommet-icons";
import LocalizedStrings from "react-localization";


const strings = new LocalizedStrings({
    en: {
        reset: "Reset",
        save: "Save"
    },
    fr: {
        reset: "Réinitialiser",
        save: "Sauvegarder"
    }
});

class Form extends React.Component {

    constructor(props) {
        super(props);
        this.state = {
            item: null,
            loading: true,
            changed: false
        };

        this.setValue = this.setValue.bind(this);
        this.setChanged = this.setChanged.bind(this);
        this.update = this.update.bind(this);
        this.save = this.save.bind(this);
        this.load = this.load.bind(this);
        this.destroy = this.destroy.bind(this);
    }

    componentDidMount() {
        this.load();
    }

    componentDidCatch(error, errorInfo) {
        this.setState({
            error: error,
            errorInfo: errorInfo
        })
    }

    componentDidUpdate(prevProps, prevState, snapshot) {
        // console.log(this.props.source, this.state);
    }

    setValue(item) {
        this.setState({
            item: item,
            changed: true
        });
    }

    setChanged(changed) {
        this.setState({changed: changed});
    }

    static sanitize(data) {
        /**
         * see:
         * https://github.com/facebook/react/issues/11417
         * https://github.com/reactjs/rfcs/pull/53
         *
         */

        Object.keys(data).forEach((key, idx) => {
            if (data[key] === null) {
                data[key] = '';
            }
        });
        //
        // if (data.hasOwnProperty('translations')) {
        //     Object.keys(data.translations).forEach((field, idx) => {
        //         Object.keys(data.translations[field]).forEach((lang, idx) => {
        //             data['translations.' + field + '.' + lang] = data.translations[field][lang];
        //         });
        //     });
        //
        // }

        return data;

    }

    load() {
        const { resource, id } = this.props.match.params;

        console.log(resource, id);
        if (!id) {
            this.setState({
                loading: false,
            });
        } else {
            console.log('load',id)
            axios.get('/api/' + resource + '/' + id)
                .then(({data}) => {
                    this.setState({
                        changed: false,
                        loading: false,
                        item: Form.sanitize(data)
                    })
                });
        }
    }

    update(value) {
        const { changed, item } = this.state;
        this.setState({
            item: value,
            changed: true
        });
    }

    save({value}) {
        const { resource, id } = this.props.match.params;
        const { history } = this.props;

        let api = '/api/' + resource;

        if (id !== undefined) {
            api += '/' + id;
        }

        this.setState({changed: false}, () =>
            axios.post(api, value)
                .then(({data}) => {
                    this.setState({
                        changed: false, item: Form.sanitize(data)
                    }, () => {
                        if (!id) {
                            history.replace('/admin/' + resource + '/' + data.id + '/edit');
                        }
                    })
                })
                .catch(() => this.setState({changed: true,})
                )
        )

    }

    destroy(fn = null) {
        const { source, id } = this.props;

        if (!id) {
            return false;
        }

        axios.delete(source + '/' + id)
            .then(() => {
                this.setState({
                    id: null,
                    item: {},
                    changed: false
                }, fn)
            });
    }

    render() {
        const { children } = this.props;
        const { resource, id } = this.props.match.params;
        const { item, changed, loading } = this.state;

        if (this.state.error) {
            return this.state.error + ' ' + this.state.errorInfo;
        }

        if (loading) {
            return '...';
        }
        if (id === undefined) {
            return (
                <GrommetForm
                    onChange={value => console.log("Change", value)}
                    onSubmit={this.save}>
                    <Box pad="medium">
                        <h2>
                            New
                        </h2>
                        {children}
                    </Box>
                    <Box direction="row" justify="start" pad="medium" gap="medium">
                        <Button primary icon={<Save/>} type="submit" label={strings.save}/>
                    </Box>
                </GrommetForm>
            );
        } else {
            return (
                <GrommetForm value={item}
                             onReset={this.load}
                             onChange={this.setValue}
                             onSubmit={this.save}>
                    <Box pad="medium">
                        <h2>
                            Modify
                        </h2>
                        {children}
                    </Box>
                    <Box direction="row" justify="start" pad={{horizontal:'medium'}} gap="medium">
                        <Button primary icon={<Save/>} disabled={!changed} type="submit" label={strings.save}/>
                    </Box>
                </GrommetForm>
            );
        }

    }
}

Form.defaultProps = {
};

export default withRouter(Form);
