(window["webpackJsonp"] = window["webpackJsonp"] || []).push([[82],{

/***/ "./resources/js/form/ui/ButtonSave.js":
/*!********************************************!*\
  !*** ./resources/js/form/ui/ButtonSave.js ***!
  \********************************************/
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





var strings = new react_localization__WEBPACK_IMPORTED_MODULE_1___default.a({
  en: {
    save: "Save"
  },
  fr: {
    save: "Sauvegarder"
  }
});

var ButtonSave = function ButtonSave(_ref) {
  var label = _ref.label;
  return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(!(function webpackMissingModule() { var e = new Error("Cannot find module '../DataSource'"); e.code = 'MODULE_NOT_FOUND'; throw e; }()), null, function (_ref2) {
    var itemChanged = _ref2.itemChanged;
    return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Button"], {
      primary: true,
      icon: /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet_icons__WEBPACK_IMPORTED_MODULE_3__["Save"], null),
      disabled: !itemChanged,
      type: "submit",
      label: /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Text"], {
        color: "white"
      }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement("strong", null, label ? label : strings.save))
    });
  });
};

/* harmony default export */ __webpack_exports__["default"] = (ButtonSave);

/***/ })

}]);