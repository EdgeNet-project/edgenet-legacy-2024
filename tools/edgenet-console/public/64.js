(window["webpackJsonp"] = window["webpackJsonp"] || []).push([[64],{

/***/ "./resources/js/form/Form.js":
/*!***********************************!*\
  !*** ./resources/js/form/Form.js ***!
  \***********************************/
/*! exports provided: default */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! react */ "./node_modules/react/index.js");
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0___default = /*#__PURE__*/__webpack_require__.n(react__WEBPACK_IMPORTED_MODULE_0__);
/* harmony import */ var react_router__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! react-router */ "./node_modules/react-router/esm/react-router.js");
/* harmony import */ var axios__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! axios */ "./node_modules/axios/index.js");
/* harmony import */ var axios__WEBPACK_IMPORTED_MODULE_2___default = /*#__PURE__*/__webpack_require__.n(axios__WEBPACK_IMPORTED_MODULE_2__);
/* harmony import */ var prop_types__WEBPACK_IMPORTED_MODULE_3__ = __webpack_require__(/*! prop-types */ "./node_modules/prop-types/index.js");
/* harmony import */ var prop_types__WEBPACK_IMPORTED_MODULE_3___default = /*#__PURE__*/__webpack_require__.n(prop_types__WEBPACK_IMPORTED_MODULE_3__);
/* harmony import */ var grommet__WEBPACK_IMPORTED_MODULE_4__ = __webpack_require__(/*! grommet */ "./node_modules/grommet/es6/index.js");
/* harmony import */ var grommet_icons__WEBPACK_IMPORTED_MODULE_5__ = __webpack_require__(/*! grommet-icons */ "./node_modules/grommet-icons/es6/index.js");
/* harmony import */ var react_localization__WEBPACK_IMPORTED_MODULE_6__ = __webpack_require__(/*! react-localization */ "./node_modules/react-localization/lib/LocalizedStrings.js");
/* harmony import */ var react_localization__WEBPACK_IMPORTED_MODULE_6___default = /*#__PURE__*/__webpack_require__.n(react_localization__WEBPACK_IMPORTED_MODULE_6__);
function _typeof(obj) { "@babel/helpers - typeof"; if (typeof Symbol === "function" && typeof Symbol.iterator === "symbol") { _typeof = function _typeof(obj) { return typeof obj; }; } else { _typeof = function _typeof(obj) { return obj && typeof Symbol === "function" && obj.constructor === Symbol && obj !== Symbol.prototype ? "symbol" : typeof obj; }; } return _typeof(obj); }

function _classCallCheck(instance, Constructor) { if (!(instance instanceof Constructor)) { throw new TypeError("Cannot call a class as a function"); } }

function _defineProperties(target, props) { for (var i = 0; i < props.length; i++) { var descriptor = props[i]; descriptor.enumerable = descriptor.enumerable || false; descriptor.configurable = true; if ("value" in descriptor) descriptor.writable = true; Object.defineProperty(target, descriptor.key, descriptor); } }

function _createClass(Constructor, protoProps, staticProps) { if (protoProps) _defineProperties(Constructor.prototype, protoProps); if (staticProps) _defineProperties(Constructor, staticProps); return Constructor; }

function _inherits(subClass, superClass) { if (typeof superClass !== "function" && superClass !== null) { throw new TypeError("Super expression must either be null or a function"); } subClass.prototype = Object.create(superClass && superClass.prototype, { constructor: { value: subClass, writable: true, configurable: true } }); if (superClass) _setPrototypeOf(subClass, superClass); }

function _setPrototypeOf(o, p) { _setPrototypeOf = Object.setPrototypeOf || function _setPrototypeOf(o, p) { o.__proto__ = p; return o; }; return _setPrototypeOf(o, p); }

function _createSuper(Derived) { return function () { var Super = _getPrototypeOf(Derived), result; if (_isNativeReflectConstruct()) { var NewTarget = _getPrototypeOf(this).constructor; result = Reflect.construct(Super, arguments, NewTarget); } else { result = Super.apply(this, arguments); } return _possibleConstructorReturn(this, result); }; }

function _possibleConstructorReturn(self, call) { if (call && (_typeof(call) === "object" || typeof call === "function")) { return call; } return _assertThisInitialized(self); }

function _assertThisInitialized(self) { if (self === void 0) { throw new ReferenceError("this hasn't been initialised - super() hasn't been called"); } return self; }

function _isNativeReflectConstruct() { if (typeof Reflect === "undefined" || !Reflect.construct) return false; if (Reflect.construct.sham) return false; if (typeof Proxy === "function") return true; try { Date.prototype.toString.call(Reflect.construct(Date, [], function () {})); return true; } catch (e) { return false; } }

function _getPrototypeOf(o) { _getPrototypeOf = Object.setPrototypeOf ? Object.getPrototypeOf : function _getPrototypeOf(o) { return o.__proto__ || Object.getPrototypeOf(o); }; return _getPrototypeOf(o); }








var strings = new react_localization__WEBPACK_IMPORTED_MODULE_6___default.a({
  en: {
    reset: "Reset",
    save: "Save"
  },
  fr: {
    reset: "Réinitialiser",
    save: "Sauvegarder"
  }
});

var Form = /*#__PURE__*/function (_React$Component) {
  _inherits(Form, _React$Component);

  var _super = _createSuper(Form);

  function Form(props) {
    var _this;

    _classCallCheck(this, Form);

    _this = _super.call(this, props);
    _this.state = {
      item: null,
      loading: true,
      changed: false
    };
    _this.setValue = _this.setValue.bind(_assertThisInitialized(_this));
    _this.setChanged = _this.setChanged.bind(_assertThisInitialized(_this));
    _this.update = _this.update.bind(_assertThisInitialized(_this));
    _this.save = _this.save.bind(_assertThisInitialized(_this));
    _this.load = _this.load.bind(_assertThisInitialized(_this));
    _this.destroy = _this.destroy.bind(_assertThisInitialized(_this));
    return _this;
  }

  _createClass(Form, [{
    key: "componentDidMount",
    value: function componentDidMount() {
      this.load();
    }
  }, {
    key: "componentDidCatch",
    value: function componentDidCatch(error, errorInfo) {
      this.setState({
        error: error,
        errorInfo: errorInfo
      });
    }
  }, {
    key: "componentDidUpdate",
    value: function componentDidUpdate(prevProps, prevState, snapshot) {// console.log(this.props.source, this.state);
    }
  }, {
    key: "setValue",
    value: function setValue(item) {
      this.setState({
        item: item,
        changed: true
      });
    }
  }, {
    key: "setChanged",
    value: function setChanged(changed) {
      this.setState({
        changed: changed
      });
    }
  }, {
    key: "load",
    value: function load() {
      var _this2 = this;

      var _this$props$match$par = this.props.match.params,
          resource = _this$props$match$par.resource,
          id = _this$props$match$par.id;
      console.log(resource, id);

      if (!id) {
        this.setState({
          loading: false
        });
      } else {
        console.log('load', id);
        axios__WEBPACK_IMPORTED_MODULE_2___default.a.get('/api/' + resource + '/' + id).then(function (_ref) {
          var data = _ref.data;

          _this2.setState({
            changed: false,
            loading: false,
            item: Form.sanitize(data)
          });
        });
      }
    }
  }, {
    key: "update",
    value: function update(value) {
      var _this$state = this.state,
          changed = _this$state.changed,
          item = _this$state.item;
      this.setState({
        item: value,
        changed: true
      });
    }
  }, {
    key: "save",
    value: function save(_ref2) {
      var _this3 = this;

      var value = _ref2.value;
      var _this$props$match$par2 = this.props.match.params,
          resource = _this$props$match$par2.resource,
          id = _this$props$match$par2.id;
      var history = this.props.history;
      var api = '/api/' + resource;

      if (id !== undefined) {
        api += '/' + id;
      }

      this.setState({
        changed: false
      }, function () {
        return axios__WEBPACK_IMPORTED_MODULE_2___default.a.post(api, value).then(function (_ref3) {
          var data = _ref3.data;

          _this3.setState({
            changed: false,
            item: Form.sanitize(data)
          }, function () {
            if (!id) {
              history.replace('/admin/' + resource + '/' + data.id + '/edit');
            }
          });
        })["catch"](function () {
          return _this3.setState({
            changed: true
          });
        });
      });
    }
  }, {
    key: "destroy",
    value: function destroy() {
      var _this4 = this;

      var fn = arguments.length > 0 && arguments[0] !== undefined ? arguments[0] : null;
      var _this$props = this.props,
          source = _this$props.source,
          id = _this$props.id;

      if (!id) {
        return false;
      }

      axios__WEBPACK_IMPORTED_MODULE_2___default.a["delete"](source + '/' + id).then(function () {
        _this4.setState({
          id: null,
          item: {},
          changed: false
        }, fn);
      });
    }
  }, {
    key: "render",
    value: function render() {
      var children = this.props.children;
      var _this$props$match$par3 = this.props.match.params,
          resource = _this$props$match$par3.resource,
          id = _this$props$match$par3.id;
      var _this$state2 = this.state,
          item = _this$state2.item,
          changed = _this$state2.changed,
          loading = _this$state2.loading;

      if (this.state.error) {
        return this.state.error + ' ' + this.state.errorInfo;
      }

      if (loading) {
        return '...';
      }

      if (id === undefined) {
        return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_4__["Form"], {
          onChange: function onChange(value) {
            return console.log("Change", value);
          },
          onSubmit: this.save
        }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_4__["Box"], {
          pad: "medium"
        }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement("h2", null, "New"), children), /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_4__["Box"], {
          direction: "row",
          justify: "start",
          pad: "medium",
          gap: "medium"
        }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_4__["Button"], {
          primary: true,
          icon: /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet_icons__WEBPACK_IMPORTED_MODULE_5__["Save"], null),
          type: "submit",
          label: strings.save
        })));
      } else {
        return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_4__["Form"], {
          value: item,
          onReset: this.load,
          onChange: this.setValue,
          onSubmit: this.save
        }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_4__["Box"], {
          pad: "medium"
        }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement("h2", null, "Modify"), children), /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_4__["Box"], {
          direction: "row",
          justify: "start",
          pad: {
            horizontal: 'medium'
          },
          gap: "medium"
        }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_4__["Button"], {
          primary: true,
          icon: /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet_icons__WEBPACK_IMPORTED_MODULE_5__["Save"], null),
          disabled: !changed,
          type: "submit",
          label: strings.save
        })));
      }
    }
  }], [{
    key: "sanitize",
    value: function sanitize(data) {
      /**
       * see:
       * https://github.com/facebook/react/issues/11417
       * https://github.com/reactjs/rfcs/pull/53
       *
       */
      Object.keys(data).forEach(function (key, idx) {
        if (data[key] === null) {
          data[key] = '';
        }
      }); //
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
  }]);

  return Form;
}(react__WEBPACK_IMPORTED_MODULE_0___default.a.Component);

Form.defaultProps = {};
/* harmony default export */ __webpack_exports__["default"] = (Object(react_router__WEBPACK_IMPORTED_MODULE_1__["withRouter"])(Form));

/***/ })

}]);