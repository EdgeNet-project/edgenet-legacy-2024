(window["webpackJsonp"] = window["webpackJsonp"] || []).push([[61],{

/***/ "./resources/js/views/NousList.js":
/*!****************************************!*\
  !*** ./resources/js/views/NousList.js ***!
  \****************************************/
/*! exports provided: default */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! react */ "./node_modules/react/index.js");
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0___default = /*#__PURE__*/__webpack_require__.n(react__WEBPACK_IMPORTED_MODULE_0__);
/* harmony import */ var grommet__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! grommet */ "./node_modules/grommet/es6/index.js");



var NousList = function NousList(_ref) {
  var item = _ref.item,
      _onClick = _ref.onClick;
  return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Box"], {
    pad: "small",
    onClick: function onClick() {
      return _onClick(item.id);
    }
  }, item.title);
};

/* harmony default export */ __webpack_exports__["default"] = (NousList);

/***/ })

}]);