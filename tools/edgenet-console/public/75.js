(window["webpackJsonp"] = window["webpackJsonp"] || []).push([[75],{

/***/ "./resources/js/data/index.js":
/*!************************************!*\
  !*** ./resources/js/data/index.js ***!
  \************************************/
/*! exports provided: Data, DataConsumer, DataContext */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony import */ var _Data__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! ./Data */ "./resources/js/data/Data.js");
/* harmony reexport (safe) */ __webpack_require__.d(__webpack_exports__, "Data", function() { return _Data__WEBPACK_IMPORTED_MODULE_0__["Data"]; });

/* harmony reexport (safe) */ __webpack_require__.d(__webpack_exports__, "DataConsumer", function() { return _Data__WEBPACK_IMPORTED_MODULE_0__["DataConsumer"]; });

/* harmony reexport (safe) */ __webpack_require__.d(__webpack_exports__, "DataContext", function() { return _Data__WEBPACK_IMPORTED_MODULE_0__["DataContext"]; });




/***/ }),

/***/ "./resources/js/data/views/List.js":
/*!*****************************************!*\
  !*** ./resources/js/data/views/List.js ***!
  \*****************************************/
/*! exports provided: default */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! react */ "./node_modules/react/index.js");
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0___default = /*#__PURE__*/__webpack_require__.n(react__WEBPACK_IMPORTED_MODULE_0__);
/* harmony import */ var grommet__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! grommet */ "./node_modules/grommet/es6/index.js");
/* harmony import */ var ___WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! ../. */ "./resources/js/data/index.js");
/* harmony import */ var _order__WEBPACK_IMPORTED_MODULE_3__ = __webpack_require__(/*! ../order */ "./resources/js/data/order/index.js");
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
        children = /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(_order__WEBPACK_IMPORTED_MODULE_3__["OrderableItem"], {
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
  //
  // let ListComponent = null;
  // if (component) {
  //     ListComponent = component;
  // } else if (children) {
  //     ListComponent = React.cloneElement(children, null);
  // } else {
  //     return null;
  // }
  return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(___WEBPACK_IMPORTED_MODULE_2__["DataConsumer"], null, function (_ref2) {
    var identifier = _ref2.identifier,
        items = _ref2.items,
        per_page = _ref2.per_page,
        itemsLoading = _ref2.itemsLoading,
        getItem = _ref2.getItem,
        currentId = _ref2.currentId,
        getItems = _ref2.getItems,
        selectable = _ref2.selectable,
        orderable = _ref2.orderable;

    if (itemsLoading) {
      return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(Loading, null);
    }

    if (items.length === 0) {
      return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Box"], {
        pad: "small",
        align: "center"
      }, "No items found");
    }

    if (currentId) {
      if (!isNaN(currentId)) {
        // check if it is a number
        currentId = parseInt(currentId);
      }
    }

    var currentIdx = null;

    if (show && currentId) {
      currentIdx = items.findIndex(function (item) {
        return item[identifier] === currentId;
      });
    }

    return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Box"], {
      overflow: "auto"
    }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["InfiniteScroll"], {
      items: items,
      onMore: getItems,
      step: per_page // show={currentIdx}
      // renderMarker={marker => itemsLoading && <Box pad="medium" background="accent-1">{marker}</Box>}

    }, function (item, j) {
      return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(ListRow, {
        key: 'items-' + j,
        item: item,
        onClick: onClick === undefined ? getItem : onClick,
        selectable: selectable,
        orderable: orderable,
        isActive: item[identifier] === currentId
      }, children);
    }));
  });
};

/* harmony default export */ __webpack_exports__["default"] = (List);

/***/ })

}]);