(window["webpackJsonp"] = window["webpackJsonp"] || []).push([[81],{

/***/ "./resources/js/form/ui/ButtonDelete.js":
/*!**********************************************!*\
  !*** ./resources/js/form/ui/ButtonDelete.js ***!
  \**********************************************/
/*! exports provided: default */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! react */ "./node_modules/react/index.js");
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0___default = /*#__PURE__*/__webpack_require__.n(react__WEBPACK_IMPORTED_MODULE_0__);
/* harmony import */ var react_localization__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! react-localization */ "./node_modules/react-localization/lib/LocalizedStrings.js");
/* harmony import */ var react_localization__WEBPACK_IMPORTED_MODULE_1___default = /*#__PURE__*/__webpack_require__.n(react_localization__WEBPACK_IMPORTED_MODULE_1__);
/* harmony import */ var grommet__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! grommet */ "./node_modules/grommet/es6/index.js");
/* harmony import */ var grommet_icons__WEBPACK_IMPORTED_MODULE_3__ = __webpack_require__(/*! grommet-icons */ "./node_modules/grommet-icons/es6/index.js");
!(function webpackMissingModule() { var e = new Error("Cannot find module '../DataSource'"); e.code = 'MODULE_NOT_FOUND'; throw e; }());
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






var strings = new react_localization__WEBPACK_IMPORTED_MODULE_1___default.a({
  en: {
    "delete": "Delete",
    cancel: "Cancel",
    confirmDelete: "Are you sure you want to delete this item?"
  },
  fr: {
    "delete": "Supprimer",
    cancel: "Anuller",
    confirmDelete: "Êtes vous sûr de vouloir supprimer définitivement cet objet ?"
  }
});

var ButtonDelete = /*#__PURE__*/function (_React$Component) {
  _inherits(ButtonDelete, _React$Component);

  var _super = _createSuper(ButtonDelete);

  function ButtonDelete(props) {
    var _this;

    _classCallCheck(this, ButtonDelete);

    _this = _super.call(this, props);
    _this.state = {
      confirmDelete: false
    };
    _this.cancel = _this.cancel.bind(_assertThisInitialized(_this));
    _this.confirm = _this.confirm.bind(_assertThisInitialized(_this));
    return _this;
  }

  _createClass(ButtonDelete, [{
    key: "cancel",
    value: function cancel() {
      this.setState({
        confirmDelete: false
      });
    }
  }, {
    key: "confirm",
    value: function confirm() {
      this.setState({
        confirmDelete: true
      });
    }
  }, {
    key: "render",
    value: function render() {
      var _this$props = this.props,
          label = _this$props.label,
          id = _this$props.id;
      var confirmDelete = this.state.confirmDelete;
      var deleteItem = this.context.deleteItem;
      if (!id) return null;
      return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Box"], {
        pad: "small"
      }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Button"], {
        icon: /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet_icons__WEBPACK_IMPORTED_MODULE_3__["Trash"], {
          color: "status-critical"
        }),
        plain: true,
        color: "status-critical",
        onClick: this.confirm,
        label: label === undefined ? strings["delete"] : label
      }), confirmDelete && /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Layer"], {
        onEsc: this.cancel,
        onClickOutside: this.cancel
      }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Box"], {
        pad: "medium"
      }, strings.confirmDelete), /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Box"], {
        direction: "row",
        justify: "center",
        gap: "small",
        pad: "medium",
        alignContent: "center"
      }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Button"], {
        label: strings.cancel,
        onClick: this.cancel
      }), /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Button"], {
        primary: true,
        label: strings["delete"],
        onClick: function onClick() {
          return deleteItem(id);
        }
      }))));
    }
  }]);

  return ButtonDelete;
}(react__WEBPACK_IMPORTED_MODULE_0___default.a.Component);

ButtonDelete.contextType = !(function webpackMissingModule() { var e = new Error("Cannot find module '../DataSource'"); e.code = 'MODULE_NOT_FOUND'; throw e; }());
/* harmony default export */ __webpack_exports__["default"] = (ButtonDelete);

/***/ })

}]);