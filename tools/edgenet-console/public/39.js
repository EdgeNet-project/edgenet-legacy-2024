(window["webpackJsonp"] = window["webpackJsonp"] || []).push([[39],{

/***/ "./resources/js/data/toolbar/Toolbar.js":
/*!**********************************************!*\
  !*** ./resources/js/data/toolbar/Toolbar.js ***!
  \**********************************************/
/*! exports provided: Toolbar, ToolbarTab, ToolbarButton */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony export (binding) */ __webpack_require__.d(__webpack_exports__, "Toolbar", function() { return Toolbar; });
/* harmony export (binding) */ __webpack_require__.d(__webpack_exports__, "ToolbarTab", function() { return ToolbarTab; });
/* harmony export (binding) */ __webpack_require__.d(__webpack_exports__, "ToolbarButton", function() { return ToolbarButton; });
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! react */ "./node_modules/react/index.js");
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0___default = /*#__PURE__*/__webpack_require__.n(react__WEBPACK_IMPORTED_MODULE_0__);
/* harmony import */ var grommet__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! grommet */ "./node_modules/grommet/es6/index.js");
/* harmony import */ var grommet_icons__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! grommet-icons */ "./node_modules/grommet-icons/es6/index.js");
function _extends() { _extends = Object.assign || function (target) { for (var i = 1; i < arguments.length; i++) { var source = arguments[i]; for (var key in source) { if (Object.prototype.hasOwnProperty.call(source, key)) { target[key] = source[key]; } } } return target; }; return _extends.apply(this, arguments); }

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





var ToolbarIcon = function ToolbarIcon(_ref) {
  var icon = _ref.icon,
      count = _ref.count;
  return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement("div", {
    style: {
      position: 'relative'
    }
  }, icon, count > 0 && /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Box"], {
    background: "brand",
    pad: {
      horizontal: 'xsmall'
    },
    style: {
      position: 'absolute',
      top: -8,
      right: -6
    },
    round: true
  }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Text"], {
    size: "xsmall"
  }, count)));
};

var ToolbarTab = /*#__PURE__*/function (_React$Component) {
  _inherits(ToolbarTab, _React$Component);

  var _super = _createSuper(ToolbarTab);

  function ToolbarTab(props) {
    var _this;

    _classCallCheck(this, ToolbarTab);

    _this = _super.call(this, props);
    _this.state = {
      hover: false
    };
    return _this;
  }

  _createClass(ToolbarTab, [{
    key: "render",
    value: function render() {
      var _this2 = this;

      var _this$props = this.props,
          _this$props$label = _this$props.label,
          label = _this$props$label === void 0 ? 'Tab' : _this$props$label,
          _this$props$icon = _this$props.icon,
          icon = _this$props$icon === void 0 ? null : _this$props$icon,
          _this$props$count = _this$props.count,
          count = _this$props$count === void 0 ? 0 : _this$props$count,
          current = _this$props.current,
          name = _this$props.name,
          onClick = _this$props.onClick;
      var hover = this.state.hover;
      var round = hover && current === null ? 'xsmall' : {
        size: "xsmall",
        corner: "top"
      };
      var background = current === name ? "light-4" : hover ? "light-2" : null;
      return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Box"], {
        pad: "small",
        round: round,
        background: background,
        onMouseEnter: function onMouseEnter() {
          return _this2.setState({
            hover: true
          });
        },
        onMouseLeave: function onMouseLeave() {
          return _this2.setState({
            hover: false
          });
        },
        onClick: onClick
      }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Button"], {
        label: label,
        plain: true,
        icon: /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(ToolbarIcon, {
          icon: icon,
          count: count
        })
      }));
    }
  }]);

  return ToolbarTab;
}(react__WEBPACK_IMPORTED_MODULE_0___default.a.Component);

var ToolbarButton = /*#__PURE__*/function (_React$Component2) {
  _inherits(ToolbarButton, _React$Component2);

  var _super2 = _createSuper(ToolbarButton);

  function ToolbarButton(props) {
    var _this3;

    _classCallCheck(this, ToolbarButton);

    _this3 = _super2.call(this, props);
    _this3.state = {
      hover: false
    };
    return _this3;
  }

  _createClass(ToolbarButton, [{
    key: "render",
    value: function render() {
      var _this4 = this;

      var hover = this.state.hover;
      var _this$props2 = this.props,
          label = _this$props2.label,
          icon = _this$props2.icon,
          active = _this$props2.active,
          onClick = _this$props2.onClick;
      return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Box"], {
        background: active ? "brand" : hover ? "light-3" : null,
        round: "small",
        pad: "xsmall",
        onMouseEnter: function onMouseEnter() {
          return _this4.setState({
            hover: true
          });
        },
        onMouseLeave: function onMouseLeave() {
          return _this4.setState({
            hover: false
          });
        },
        onClick: onClick
      }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Button"], {
        icon: icon,
        label: label,
        plain: true
      }));
    }
  }]);

  return ToolbarButton;
}(react__WEBPACK_IMPORTED_MODULE_0___default.a.Component);

var Toolbar = /*#__PURE__*/function (_React$PureComponent) {
  _inherits(Toolbar, _React$PureComponent);

  var _super3 = _createSuper(Toolbar);

  function Toolbar(props) {
    var _this5;

    _classCallCheck(this, Toolbar);

    _this5 = _super3.call(this, props);
    _this5.state = {
      showTool: null,
      error: ''
    };
    _this5.handleToolbarTab = _this5.handleToolbarTab.bind(_assertThisInitialized(_this5));
    return _this5;
  }

  _createClass(Toolbar, [{
    key: "componentDidMount",
    value: function componentDidMount() {}
  }, {
    key: "componentDidCatch",
    value: function componentDidCatch(error, errorInfo) {
      this.setState({
        error: error
      });
    }
  }, {
    key: "handleToolbarTab",
    value: function handleToolbarTab(name) {
      this.setState({
        showTool: this.state.showTool === name ? null : name
      });
    }
  }, {
    key: "render",
    value: function render() {
      var _this6 = this;

      var children = this.props.children;
      var _this$state = this.state,
          showTool = _this$state.showTool,
          error = _this$state.error;

      if (error) {
        return error;
      }

      var currentTool = null;
      var toolbar = react__WEBPACK_IMPORTED_MODULE_0___default.a.Children.map(children, function (child) {
        var _child$type = child.type,
            Tab = _child$type.Tab,
            Button = _child$type.Button,
            name = _child$type.name;

        if (showTool === name) {
          currentTool = /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Box"], {
            background: "light-4"
          }, child);
        }

        if (Tab !== undefined) {
          return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(Tab, _extends({
            name: name,
            current: showTool,
            onClick: function onClick() {
              return _this6.handleToolbarTab(name);
            }
          }, child.props));
        }

        if (Button !== undefined) {
          return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(Button, null);
        }

        return child;
      });
      return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Box"], {
        flex: false,
        margin: {
          vertical: 'xsmall'
        }
      }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Box"], {
        direction: "row"
      }, toolbar), currentTool);
    }
  }]);

  return Toolbar;
}(react__WEBPACK_IMPORTED_MODULE_0___default.a.PureComponent);



/***/ })

}]);