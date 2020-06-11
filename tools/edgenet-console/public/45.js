(window["webpackJsonp"] = window["webpackJsonp"] || []).push([[45],{

/***/ "./resources/js/form/old/FormFieldDate.js":
/*!************************************************!*\
  !*** ./resources/js/form/old/FormFieldDate.js ***!
  \************************************************/
/*! exports provided: default */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! react */ "./node_modules/react/index.js");
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0___default = /*#__PURE__*/__webpack_require__.n(react__WEBPACK_IMPORTED_MODULE_0__);
/* harmony import */ var grommet__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! grommet */ "./node_modules/grommet/es6/index.js");
/* harmony import */ var grommet_icons__WEBPACK_IMPORTED_MODULE_2__ = __webpack_require__(/*! grommet-icons */ "./node_modules/grommet-icons/es6/index.js");
function _slicedToArray(arr, i) { return _arrayWithHoles(arr) || _iterableToArrayLimit(arr, i) || _unsupportedIterableToArray(arr, i) || _nonIterableRest(); }

function _nonIterableRest() { throw new TypeError("Invalid attempt to destructure non-iterable instance.\nIn order to be iterable, non-array objects must have a [Symbol.iterator]() method."); }

function _unsupportedIterableToArray(o, minLen) { if (!o) return; if (typeof o === "string") return _arrayLikeToArray(o, minLen); var n = Object.prototype.toString.call(o).slice(8, -1); if (n === "Object" && o.constructor) n = o.constructor.name; if (n === "Map" || n === "Set") return Array.from(n); if (n === "Arguments" || /^(?:Ui|I)nt(?:8|16|32)(?:Clamped)?Array$/.test(n)) return _arrayLikeToArray(o, minLen); }

function _arrayLikeToArray(arr, len) { if (len == null || len > arr.length) len = arr.length; for (var i = 0, arr2 = new Array(len); i < len; i++) { arr2[i] = arr[i]; } return arr2; }

function _iterableToArrayLimit(arr, i) { if (typeof Symbol === "undefined" || !(Symbol.iterator in Object(arr))) return; var _arr = []; var _n = true; var _d = false; var _e = undefined; try { for (var _i = arr[Symbol.iterator](), _s; !(_n = (_s = _i.next()).done); _n = true) { _arr.push(_s.value); if (i && _arr.length === i) break; } } catch (err) { _d = true; _e = err; } finally { try { if (!_n && _i["return"] != null) _i["return"](); } finally { if (_d) throw _e; } } return _arr; }

function _arrayWithHoles(arr) { if (Array.isArray(arr)) return arr; }





var DropContent = function DropContent(_ref) {
  var initialDate = _ref.date,
      initialTime = _ref.time,
      onClose = _ref.onClose;

  var _React$useState = react__WEBPACK_IMPORTED_MODULE_0___default.a.useState(),
      _React$useState2 = _slicedToArray(_React$useState, 2),
      date = _React$useState2[0],
      setDate = _React$useState2[1];

  var _React$useState3 = react__WEBPACK_IMPORTED_MODULE_0___default.a.useState(),
      _React$useState4 = _slicedToArray(_React$useState3, 2),
      time = _React$useState4[0],
      setTime = _React$useState4[1];

  var close = function close() {
    return onClose(date || initialDate, time || initialTime);
  };

  return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Box"], {
    align: "center"
  }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Calendar"], {
    animate: false,
    date: date || initialDate,
    onSelect: setDate,
    showAdjacentDays: false
  }), /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Box"], {
    flex: false,
    pad: "medium",
    gap: "medium"
  }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Keyboard"], {
    onEnter: function onEnter(event) {
      event.preventDefault(); // so drop doesn't re-open

      close();
    }
  }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["MaskedInput"], {
    mask: [{
      length: [1, 2],
      options: ["1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12"],
      regexp: /^1[1-2]$|^[0-9]$/,
      placeholder: "hh"
    }, {
      fixed: ":"
    }, {
      length: 2,
      options: ["00", "15", "30", "45"],
      regexp: /^[0-5][0-9]$|^[0-9]$/,
      placeholder: "mm"
    }, {
      fixed: " "
    }, {
      length: 2,
      options: ["am", "pm"],
      regexp: /^[ap]m$|^[AP]M$|^[aApP]$/,
      placeholder: "ap"
    }],
    value: time || initialTime,
    name: "maskedInput",
    onChange: function onChange(event) {
      return setTime(event.target.value);
    }
  })), /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Box"], {
    flex: false
  }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Button"], {
    label: "Done",
    onClick: close
  }))));
};

var FormFieldDate = function FormFieldDate() {
  var _React$useState5 = react__WEBPACK_IMPORTED_MODULE_0___default.a.useState(),
      _React$useState6 = _slicedToArray(_React$useState5, 2),
      date = _React$useState6[0],
      setDate = _React$useState6[1];

  var _React$useState7 = react__WEBPACK_IMPORTED_MODULE_0___default.a.useState(""),
      _React$useState8 = _slicedToArray(_React$useState7, 2),
      time = _React$useState8[0],
      setTime = _React$useState8[1];

  var _React$useState9 = react__WEBPACK_IMPORTED_MODULE_0___default.a.useState(),
      _React$useState10 = _slicedToArray(_React$useState9, 2),
      open = _React$useState10[0],
      setOpen = _React$useState10[1];

  var onClose = function onClose(nextDate, nextTime) {
    setDate(nextDate);
    setTime(nextTime);
    setOpen(false);
    setTimeout(function () {
      return setOpen(undefined);
    }, 1);
  };

  return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["DropButton"], {
    open: open,
    onClose: function onClose() {
      return setOpen(false);
    },
    onOpen: function onOpen() {
      return setOpen(true);
    },
    dropContent: /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(DropContent, {
      date: date,
      time: time,
      onClose: onClose
    })
  }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Box"], {
    direction: "row",
    gap: "medium",
    align: "center",
    pad: "small"
  }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Text"], {
    color: date ? undefined : "dark-5"
  }, date ? "".concat(new Date(date).toLocaleDateString(), " ").concat(time) : "Select date & time"), /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet_icons__WEBPACK_IMPORTED_MODULE_2__["Schedule"], null)));
};

/* harmony default export */ __webpack_exports__["default"] = (FormFieldDate);

/***/ })

}]);