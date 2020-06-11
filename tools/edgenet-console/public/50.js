(window["webpackJsonp"] = window["webpackJsonp"] || []).push([[50],{

/***/ "./resources/js/form/old/FormFieldVideo.js":
/*!*************************************************!*\
  !*** ./resources/js/form/old/FormFieldVideo.js ***!
  \*************************************************/
/*! exports provided: default */
/***/ (function(module, __webpack_exports__, __webpack_require__) {

"use strict";
__webpack_require__.r(__webpack_exports__);
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0__ = __webpack_require__(/*! react */ "./node_modules/react/index.js");
/* harmony import */ var react__WEBPACK_IMPORTED_MODULE_0___default = /*#__PURE__*/__webpack_require__.n(react__WEBPACK_IMPORTED_MODULE_0__);
/* harmony import */ var grommet__WEBPACK_IMPORTED_MODULE_1__ = __webpack_require__(/*! grommet */ "./node_modules/grommet/es6/index.js");
!(function webpackMissingModule() { var e = new Error("Cannot find module '../DataSource'"); e.code = 'MODULE_NOT_FOUND'; throw e; }());


 // TOFIX

var FormFieldVideo = function FormFieldVideo(_ref) {
  var name = _ref.name;
  return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(!(function webpackMissingModule() { var e = new Error("Cannot find module '../DataSource'"); e.code = 'MODULE_NOT_FOUND'; throw e; }()), null, function (_ref2) {
    var item = _ref2.item;
    return item[name] ? /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_1__["Image"], {
      src: item[name]
    }) : null;
  });
};

/* harmony default export */ __webpack_exports__["default"] = (FormFieldVideo);

/***/ })

}]);