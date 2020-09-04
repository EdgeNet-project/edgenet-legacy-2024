(window["webpackJsonp"] = window["webpackJsonp"] || []).push([[1],{

/***/ "./resources/js/modules/k8s/K8s.js":
/*!*****************************************!*\
  !*** ./resources/js/modules/k8s/K8s.js ***!
  \*****************************************/
/*! exports provided: K8s, K8sContext, K8sConsumer */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export (binding) */ __webpack_require__.d(__webpack_exports__, "K8s", function() { return K8s; });
/* harmony export (binding) */ __webpack_require__.d(__webpack_exports__, "K8sContext", function() { return K8sContext; });
/* harmony export (binding) */ __webpack_require__.d(__webpack_exports__, "K8sConsumer", function() { return K8sConsumer; });
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! react */ "./node_modules/react/index.js");
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0___default = /*#__PURE__*/__webpack_require__.n(react__WEBPACK_IMPORTED_MODULE_0__);
/* harmony import */ var prop_types__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! prop-types */ "./node_modules/prop-types/index.js");
/* harmony import */ var prop_types__WEBPACK_IMPORTED_MODULE_1___default = /*#__PURE__*/__webpack_require__.n(prop_types__WEBPACK_IMPORTED_MODULE_1__);
/* harmony import */ var axios__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! axios */ "./node_modules/axios/index.js");
/* harmony import */ var axios__WEBPACK_IMPORTED_MODULE_2___default = /*#__PURE__*/__webpack_require__.n(axios__WEBPACK_IMPORTED_MODULE_2__);
/* harmony import */ var qs__WEBPACK_IMPORTED_MODULE_3__ = __webpack_require__(/*! qs */ "./node_modules/qs/lib/index.js");
/* harmony import */ var qs__WEBPACK_IMPORTED_MODULE_3___default = /*#__PURE__*/__webpack_require__.n(qs__WEBPACK_IMPORTED_MODULE_3__);
function _typeof(obj) { "@babel/helpers - typeof"; if (typeof Symbol === "function" && typeof Symbol.iterator === "symbol") { _typeof = function _typeof(obj) { return typeof obj; }; } else { _typeof = function _typeof(obj) { return obj && typeof Symbol === "function" && obj.constructor === Symbol && obj !== Symbol.prototype ? "symbol" : typeof obj; }; } return _typeof(obj); }

function ownKeys(object, enumerableOnly) { var keys = Object.keys(object); if (Object.getOwnPropertySymbols) { var symbols = Object.getOwnPropertySymbols(object); if (enumerableOnly) symbols = symbols.filter(function (sym) { return Object.getOwnPropertyDescriptor(object, sym).enumerable; }); keys.push.apply(keys, symbols); } return keys; }

function _objectSpread(target) { for (var i = 1; i < arguments.length; i++) { var source = arguments[i] != null ? arguments[i] : {}; if (i % 2) { ownKeys(Object(source), true).forEach(function (key) { _defineProperty(target, key, source[key]); }); } else if (Object.getOwnPropertyDescriptors) { Object.defineProperties(target, Object.getOwnPropertyDescriptors(source)); } else { ownKeys(Object(source)).forEach(function (key) { Object.defineProperty(target, key, Object.getOwnPropertyDescriptor(source, key)); }); } } return target; }

function _defineProperty(obj, key, value) { if (key in obj) { Object.defineProperty(obj, key, { value: value, enumerable: true, configurable: true, writable: true }); } else { obj[key] = value; } return obj; }

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





var K8sContext = react__WEBPACK_IMPORTED_MODULE_0___default.a.createContext({});
var K8sConsumer = K8sContext.Consumer;

var K8s = /*#__PURE__*/function (_React$Component) {
  _inherits(K8s, _React$Component);

  var _super = _createSuper(K8s);

  function K8s(props) {
    var _this;

    _classCallCheck(this, K8s);

    _this = _super.call(this, props);
    _this.state = {
      resource: null,
      name: null,
      items: [],
      metadata: [],
      queryParams: {},
      loading: true
    };
    _this.get = _this.get.bind(_assertThisInitialized(_this));
    _this.push = _this.push.bind(_assertThisInitialized(_this));
    _this.pull = _this.pull.bind(_assertThisInitialized(_this));
    return _this;
  }

  _createClass(K8s, [{
    key: "componentDidMount",
    value: function componentDidMount() {
      // const { url, id, sort_by, filter, limit } = this.props;
      // this.setState({
      //     source: source,
      //     id: id
      // });
      // id ? this.getItem(id) : this.setQueryParams({
      //     sort_by: sort_by,
      //     // filter: filter,
      //     // limit: limit
      // });
      this.get();
    }
  }, {
    key: "componentDidUpdate",
    value: function componentDidUpdate(prevProps, prevState, snapshot) {
      // console.log('## ListContext update: ', prevState, '=>', this.state)
      // console.log('## ListContext update: ', prevProps, '=>', this.props)
      //
      var _this$props = this.props,
          resource = _this$props.resource,
          id = _this$props.id;

      if (prevProps.resource !== resource) {
        // reloading items
        this.get();
      }

      if (prevProps.id !== id) {
        // reloading item
        this.get(id);
      }
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
    key: "componentWillUnmount",
    value: function componentWillUnmount() {}
    /**
     * Sets the query params for the list request.
     */

  }, {
    key: "setQueryParams",
    value: function setQueryParams(params) {
      var queryParams = {};

      if (params.sort_by !== undefined) {
        queryParams = {
          sort_by: Object.fromEntries(params.sort_by.map(function (s) {
            return [s.name, s.direction];
          }))
        };
      }

      if (params.filter !== undefined) {
        queryParams = _objectSpread({}, queryParams, {
          filter: params.filter
        });
      } // if (params.limit !== undefined) {
      //     queryParams = {
      //         ...queryParams, limit: params.limit
      //     };
      // }


      if (params.search !== undefined) {
        queryParams = _objectSpread({}, queryParams, {
          search: params.search
        });
      }

      this.setState({
        items: [],
        queryParams: _objectSpread({}, this.state.queryParams, {}, queryParams)
      }, this.refreshItems);
    }
  }, {
    key: "get",
    value: function get() {
      var _this2 = this;

      var api = this.props.api;
      var _this$state = this.state,
          items = _this$state.items,
          current_page = _this$state.current_page,
          last_page = _this$state.last_page,
          queryParams = _this$state.queryParams;
      if (!api) return false; // if (current_page >= last_page) return;

      axios__WEBPACK_IMPORTED_MODULE_2___default.a.get(api.server + api.url, {
        params: _objectSpread({}, queryParams, {
          page: current_page + 1
        }),
        paramsSerializer: qs__WEBPACK_IMPORTED_MODULE_3___default.a.stringify
      }).then(function (_ref) {
        var data = _ref.data;

        _this2.setState(_objectSpread({}, data, {
          loading: false
        }));
      })["catch"](function (error) {
        console.log(error);
      });
    }
  }, {
    key: "push",
    value: function push(item) {
      var items = this.state.items;
      this.setState({
        items: items.concat([item])
      });
    }
  }, {
    key: "pull",
    value: function pull(item) {
      var items = this.state.items;
      this.setState({
        items: items.filter(function (i) {
          return i.id !== item.id;
        })
      });
    }
  }, {
    key: "refresh",
    value: function refresh() {
      this.setState({
        items: [],
        metadata: [],
        loading: true
      }, this.get);
    }
  }, {
    key: "render",
    value: function render() {
      var children = this.props.children;
      var resource = this.props.resource;
      var _this$state2 = this.state,
          items = _this$state2.items,
          metadata = _this$state2.metadata,
          loading = _this$state2.loading,
          error = _this$state2.error,
          errorInfo = _this$state2.errorInfo;

      if (error) {
        return error + ' ' + errorInfo;
      }

      return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(K8sContext.Provider, {
        value: {
          resource: resource,
          items: items,
          loading: loading,
          get: this.get,
          push: this.pushItem,
          pull: this.pullItem
        }
      }, children);
    }
  }], [{
    key: "getDerivedStateFromProps",
    value: function getDerivedStateFromProps(props, state) {
      if (props.resource !== state.resource || props.id !== state.id) {
        return {
          resource: props.resource,
          id: props.id,
          items: [],
          current_page: 0,
          last_page: 1,
          per_page: null,
          total: 0,
          queryParams: {},
          loading: true
        };
      }

      return null;
    }
  }, {
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
      });
      return data;
    }
  }]);

  return K8s;
}(react__WEBPACK_IMPORTED_MODULE_0___default.a.Component);

K8s.propTypes = {
  api: prop_types__WEBPACK_IMPORTED_MODULE_1___default.a.object.isRequired,
  id: prop_types__WEBPACK_IMPORTED_MODULE_1___default.a.any
};
K8s.defaultProps = {};


/***/ }),

/***/ "./resources/js/modules/k8s/index.js":
/*!*******************************************!*\
  !*** ./resources/js/modules/k8s/index.js ***!
  \*******************************************/
/*! exports provided: K8s, K8sContext, K8sConsumer */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony import */ var _K8s__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ./K8s */ "./resources/js/modules/k8s/K8s.js");
/* harmony reexport (safe) */ __webpack_require__.d(__webpack_exports__, "K8s", function() { return _K8s__WEBPACK_IMPORTED_MODULE_0__["K8s"]; });

/* harmony reexport (safe) */ __webpack_require__.d(__webpack_exports__, "K8sContext", function() { return _K8s__WEBPACK_IMPORTED_MODULE_0__["K8sContext"]; });

/* harmony reexport (safe) */ __webpack_require__.d(__webpack_exports__, "K8sConsumer", function() { return _K8s__WEBPACK_IMPORTED_MODULE_0__["K8sConsumer"]; });




/***/ }),

/***/ "./resources/js/modules/k8s/ui/List.js":
/*!*********************************************!*\
  !*** ./resources/js/modules/k8s/ui/List.js ***!
  \*********************************************/
/*! exports provided: default */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! react */ "./node_modules/react/index.js");
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0___default = /*#__PURE__*/__webpack_require__.n(react__WEBPACK_IMPORTED_MODULE_0__);
/* harmony import */ var grommet__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! grommet */ "./node_modules/grommet/es6/index.js");
/* harmony import */ var ___WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! ../. */ "./resources/js/modules/k8s/index.js");
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





var Loading = function Loading() {
  return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Box"], {
    flex: "grow",
    justify: "center",
    align: "center"
  }, "...");
};

var ListRow = /*#__PURE__*/function (_React$Component) {
  _inherits(ListRow, _React$Component);

  var _super = _createSuper(ListRow);

  function ListRow(props) {
    var _this;

    _classCallCheck(this, ListRow);

    _this = _super.call(this, props);
    _this.state = {
      isMouseOver: false
    };
    return _this;
  }

  _createClass(ListRow, [{
    key: "render",
    value: function render() {
      var _this2 = this;

      var children = this.props.children;
      var _this$props = this.props,
          item = _this$props.item,
          isActive = _this$props.isActive,
          orderable = _this$props.orderable,
          _onClick = _this$props.onClick;
      var isMouseOver = this.state.isMouseOver;
      var background = isActive ? 'light-4' : isMouseOver ? 'light-2' : 'light-1';
      children = react__WEBPACK_IMPORTED_MODULE_0___default.a.cloneElement(children, {
        item: item,
        isActive: isActive,
        isMouseOver: isMouseOver
      }, null);

      if (orderable) {
        children = /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(OrderableItem, {
          isMouseOver: isMouseOver,
          item: item
        }, children);
      }

      return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Box"], {
        onMouseEnter: function onMouseEnter() {
          return _this2.setState({
            isMouseOver: true
          });
        },
        onMouseLeave: function onMouseLeave() {
          return _this2.setState({
            isMouseOver: false
          });
        },
        onClick: function onClick() {
          return _onClick(item);
        },
        background: background,
        border: {
          side: 'bottom',
          color: 'light-4'
        },
        flex: false
      }, children);
    }
  }]);

  return ListRow;
}(react__WEBPACK_IMPORTED_MODULE_0___default.a.Component);

var List = function List(_ref) {
  var children = _ref.children,
      onClick = _ref.onClick,
      _ref$show = _ref.show,
      show = _ref$show === void 0 ? false : _ref$show;
  return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(___WEBPACK_IMPORTED_MODULE_2__["K8sConsumer"], null, function (_ref2) {
    var items = _ref2.items,
        per_page = _ref2.per_page,
        loading = _ref2.loading,
        get = _ref2.get;

    if (loading) {
      return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(Loading, null);
    }

    if (items.length === 0) {
      return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Box"], {
        pad: "small",
        align: "center"
      }, "No resources found");
    } // if (currentId) {
    //     if (!isNaN(currentId)) {
    //         // check if it is a number
    //         currentId = parseInt(currentId);
    //     }
    // }
    // let currentIdx = null;
    // if (show && currentId) {
    //     currentIdx = items.findIndex(item => item[identifier] === currentId)
    // }


    return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Box"], {
      overflow: "auto"
    }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["InfiniteScroll"], {
      items: items // onMore={get}
      ,
      step: per_page // show={currentIdx}
      // renderMarker={marker => itemsLoading && <Box pad="medium" background="accent-1">{marker}</Box>}

    }, function (item, j) {
      return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(ListRow, {
        key: 'items-' + j,
        item: item
      }, children);
    }));
  });
};

/* harmony default export */ __webpack_exports__["default"] = (List);

/***/ })

}]);