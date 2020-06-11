(window["webpackJsonp"] = window["webpackJsonp"] || []).push([[37],{

/***/ "./resources/js/data/order/OrderableItemNative.js":
/*!********************************************************!*\
  !*** ./resources/js/data/order/OrderableItemNative.js ***!
  \********************************************************/
/*! exports provided: default */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! react */ "./node_modules/react/index.js");
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0___default = /*#__PURE__*/__webpack_require__.n(react__WEBPACK_IMPORTED_MODULE_0__);
/* harmony import */ var prop_types__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! prop-types */ "./node_modules/prop-types/index.js");
/* harmony import */ var prop_types__WEBPACK_IMPORTED_MODULE_1___default = /*#__PURE__*/__webpack_require__.n(prop_types__WEBPACK_IMPORTED_MODULE_1__);
/* harmony import */ var grommet__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! grommet */ "./node_modules/grommet/es6/index.js");
/* harmony import */ var _Orderable__WEBPACK_IMPORTED_MODULE_3__ = __webpack_require__(/*! ./Orderable */ "./resources/js/data/order/Orderable.js");
/* harmony import */ var grommet_icons_es6__WEBPACK_IMPORTED_MODULE_4__ = __webpack_require__(/*! grommet-icons/es6 */ "./node_modules/grommet-icons/es6/index.js");
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







var OrderableItem = /*#__PURE__*/function (_React$Component) {
  _inherits(OrderableItem, _React$Component);

  var _super = _createSuper(OrderableItem);

  function OrderableItem(props) {
    var _this;

    _classCallCheck(this, OrderableItem);

    _this = _super.call(this, props);
    _this.state = {
      cursor: 'auto',
      draggable: false
    };
    _this.setCursorGrab = _this.setCursorGrab.bind(_assertThisInitialized(_this));
    _this.setCursorGrabbing = _this.setCursorGrabbing.bind(_assertThisInitialized(_this));
    _this.setCursorAuto = _this.setCursorAuto.bind(_assertThisInitialized(_this));
    return _this;
  }

  _createClass(OrderableItem, [{
    key: "setCursorGrab",
    value: function setCursorGrab() {
      this.setState({
        cursor: 'grab',
        draggable: false
      });
    }
  }, {
    key: "setCursorGrabbing",
    value: function setCursorGrabbing() {
      this.setState({
        cursor: 'grabbing',
        draggable: true
      });
    }
  }, {
    key: "setCursorAuto",
    value: function setCursorAuto() {
      this.setState({
        cursor: 'auto'
      });
    }
  }, {
    key: "render",
    value: function render() {
      var _this2 = this;

      var _this$props = this.props,
          children = _this$props.children,
          item = _this$props.item,
          isMouseOver = _this$props.isMouseOver;
      var _this$state = this.state,
          cursor = _this$state.cursor,
          draggable = _this$state.draggable;
      return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(_Orderable__WEBPACK_IMPORTED_MODULE_3__["OrderableConsumer"], null, function (_ref) {
        var onDragStart = _ref.onDragStart,
            _onDragOver = _ref.onDragOver,
            onDragEnd = _ref.onDragEnd,
            elementOver = _ref.elementOver;
        return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Box"], {
          draggable: draggable,
          onDragStart: onDragStart,
          onDragOver: function onDragOver(ev) {
            return _onDragOver(ev, item.id);
          },
          onDragEnd: onDragEnd,
          border: {
            side: 'top',
            size: 'small',
            color: elementOver === item.id ? 'brand' : 'inherit'
          },
          direction: "row",
          background: draggable ? 'white' : ''
        }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Box"], {
          justify: "center",
          align: "center",
          style: {
            cursor: cursor
          },
          pad: {
            right: "xsmall"
          }
        }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet_icons_es6__WEBPACK_IMPORTED_MODULE_4__["Drag"], {
          onMouseOver: _this2.setCursorGrab,
          onMouseOut: _this2.setCursorAuto,
          onMouseDown: _this2.setCursorGrabbing,
          onMouseUp: _this2.setCursorGrab
        })), children);
      });
    }
  }]);

  return OrderableItem;
}(react__WEBPACK_IMPORTED_MODULE_0___default.a.Component);

OrderableItem.propTypes = {// item: PropTypes.object.isRequired
};
OrderableItem.defaultProps = {};
/* harmony default export */ __webpack_exports__["default"] = (OrderableItem);

/***/ })

}]);