(window["webpackJsonp"] = window["webpackJsonp"] || []).push([[62],{

/***/ "./resources/js/core/ApplicationSetup.js":
/*!***********************************************!*\
  !*** ./resources/js/core/ApplicationSetup.js ***!
  \***********************************************/
/*! exports provided: ApplicationSetup, ApplicationContext, ApplicationConsumer */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export (binding) */ __webpack_require__.d(__webpack_exports__, "ApplicationSetup", function() { return ApplicationSetup; });
/* harmony export (binding) */ __webpack_require__.d(__webpack_exports__, "ApplicationContext", function() { return ApplicationContext; });
/* harmony export (binding) */ __webpack_require__.d(__webpack_exports__, "ApplicationConsumer", function() { return ApplicationConsumer; });
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! react */ "./node_modules/react/index.js");
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0___default = /*#__PURE__*/__webpack_require__.n(react__WEBPACK_IMPORTED_MODULE_0__);
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


var ApplicationContext = react__WEBPACK_IMPORTED_MODULE_0___default.a.createContext({
  resources: []
});
var ApplicationConsumer = ApplicationContext.Consumer;

var ApplicationSetup = /*#__PURE__*/function (_React$Component) {
  _inherits(ApplicationSetup, _React$Component);

  var _super = _createSuper(ApplicationSetup);

  function ApplicationSetup(props) {
    var _this;

    _classCallCheck(this, ApplicationSetup);

    _this = _super.call(this, props);
    _this.state = {};
    return _this;
  }

  _createClass(ApplicationSetup, [{
    key: "componentDidMount",
    value: function componentDidMount() {}
  }, {
    key: "render",
    value: function render() {
      var _this$props = this.props,
          children = _this$props.children,
          menu = _this$props.menu,
          resources = _this$props.resources;
      return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(ApplicationContext.Provider, {
        value: {
          menu: menu,
          resources: resources
        }
      }, children);
    }
  }]);

  return ApplicationSetup;
}(react__WEBPACK_IMPORTED_MODULE_0___default.a.Component);



/***/ }),

/***/ "./resources/js/form/Related.js":
/*!**************************************!*\
  !*** ./resources/js/form/Related.js ***!
  \**************************************/
/*! exports provided: default */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! react */ "./node_modules/react/index.js");
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0___default = /*#__PURE__*/__webpack_require__.n(react__WEBPACK_IMPORTED_MODULE_0__);
/* harmony import */ var grommet__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! grommet */ "./node_modules/grommet/es6/index.js");
/* harmony import */ var _core_ApplicationSetup__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! ../core/ApplicationSetup */ "./resources/js/core/ApplicationSetup.js");
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

function _objectWithoutProperties(source, excluded) { if (source == null) return {}; var target = _objectWithoutPropertiesLoose(source, excluded); var key, i; if (Object.getOwnPropertySymbols) { var sourceSymbolKeys = Object.getOwnPropertySymbols(source); for (i = 0; i < sourceSymbolKeys.length; i++) { key = sourceSymbolKeys[i]; if (excluded.indexOf(key) >= 0) continue; if (!Object.prototype.propertyIsEnumerable.call(source, key)) continue; target[key] = source[key]; } } return target; }

function _objectWithoutPropertiesLoose(source, excluded) { if (source == null) return {}; var target = {}; var sourceKeys = Object.keys(source); var key, i; for (i = 0; i < sourceKeys.length; i++) { key = sourceKeys[i]; if (excluded.indexOf(key) >= 0) continue; target[key] = source[key]; } return target; }





var ItemsTab = function ItemsTab(props) {
  var Component = react__WEBPACK_IMPORTED_MODULE_0___default.a.lazy(function () {
    return Promise.all(/*! import() */[__webpack_require__.e(0), __webpack_require__.e(1), __webpack_require__.e(2), __webpack_require__.e(13), __webpack_require__.e(65)]).then(__webpack_require__.bind(null, /*! ./tabs/Items */ "./resources/js/form/tabs/Items.js"))["catch"](function () {
      return {
        "default": function _default() {
          return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement("div", null, "Not found");
        }
      };
    });
  });
  return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(react__WEBPACK_IMPORTED_MODULE_0__["Suspense"], {
    fallback: /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement("div", null, "Loading...")
  }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(Component, props));
};

var MediaTab = function MediaTab(_ref) {
  var component = _ref.component,
      props = _objectWithoutProperties(_ref, ["component"]);

  var Component = react__WEBPACK_IMPORTED_MODULE_0___default.a.lazy(function () {
    return __webpack_require__("./resources/js/form/tabs lazy recursive ^\\.\\/.*$")("./" + component.charAt(0).toUpperCase() + component.slice(1))["catch"](function () {
      return {
        "default": function _default() {
          return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement("div", null, "Not found");
        }
      };
    });
  });
  return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(react__WEBPACK_IMPORTED_MODULE_0__["Suspense"], {
    fallback: /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement("div", null, "Loading...")
  }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(Component, props));
};

var Related = /*#__PURE__*/function (_React$Component) {
  _inherits(Related, _React$Component);

  var _super = _createSuper(Related);

  function Related(props) {
    var _this;

    _classCallCheck(this, Related);

    _this = _super.call(this, props);
    _this.state = {
      index: 0,
      media: [],
      related: []
    };
    return _this;
  }

  _createClass(Related, [{
    key: "componentDidMount",
    value: function componentDidMount() {
      var resource = this.props.match.params.resource;
      var resources = this.context.resources;
      var r = resources.find(function (r) {
        return r.name === resource;
      });
      this.setState({
        media: Array.isArray(r.media) && r.media.length > 0 ? r.media : [],
        related: Array.isArray(r.related) && r.related.length > 0 ? r.related : []
      });
    }
  }, {
    key: "render",
    value: function render() {
      var _this2 = this;

      var _this$props$match$par = this.props.match.params,
          resource = _this$props$match$par.resource,
          id = _this$props$match$par.id;
      var _this$state = this.state,
          index = _this$state.index,
          media = _this$state.media,
          related = _this$state.related;
      if (!media.length && !related.length) return null;
      return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Tabs"], {
        activeIndex: index,
        onActive: function onActive(index) {
          return _this2.setState({
            index: index
          });
        }
      }, media.map(function (m) {
        return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Tab"], {
          key: m,
          title: MediaTab.title || m
        }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Box"], {
          pad: "medium"
        }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(MediaTab, {
          component: m,
          resource: resource,
          id: id
        })));
      }), related.map(function (m) {
        return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Tab"], {
          key: m,
          title: ItemsTab.title || m
        }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Box"], {
          pad: "medium"
        }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(ItemsTab, {
          related: m,
          resource: resource,
          id: id
        })));
      }));
    }
  }]);

  return Related;
}(react__WEBPACK_IMPORTED_MODULE_0___default.a.Component);

Related.contextType = _core_ApplicationSetup__WEBPACK_IMPORTED_MODULE_2__["ApplicationContext"];
/* harmony default export */ __webpack_exports__["default"] = (Related);

/***/ }),

/***/ "./resources/js/form/tabs lazy recursive ^\\.\\/.*$":
/*!***************************************************************!*\
  !*** ./resources/js/form/tabs lazy ^\.\/.*$ namespace object ***!
  \***************************************************************/
/*! no static exports found */
/***/ (function(module, exports, __webpack_require__) {

var map = {
	"./Images": [
		"./resources/js/form/tabs/Images.js",
		0,
		1,
		4,
		2,
		13,
		66
	],
	"./Images.js": [
		"./resources/js/form/tabs/Images.js",
		0,
		1,
		4,
		2,
		13,
		66
	],
	"./Items": [
		"./resources/js/form/tabs/Items.js",
		0,
		1,
		2,
		13,
		65
	],
	"./Items.js": [
		"./resources/js/form/tabs/Items.js",
		0,
		1,
		2,
		13,
		65
	]
};
function webpackAsyncContext(req) {
	if(!__webpack_require__.o(map, req)) {
		return Promise.resolve().then(function() {
			var e = new Error("Cannot find module '" + req + "'");
			e.code = 'MODULE_NOT_FOUND';
			throw e;
		});
	}

	var ids = map[req], id = ids[0];
	return Promise.all(ids.slice(1).map(__webpack_require__.e)).then(function() {
		return __webpack_require__(id);
	});
}
webpackAsyncContext.keys = function webpackAsyncContextKeys() {
	return Object.keys(map);
};
webpackAsyncContext.id = "./resources/js/form/tabs lazy recursive ^\\.\\/.*$";
module.exports = webpackAsyncContext;

/***/ })

}]);