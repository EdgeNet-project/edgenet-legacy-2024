(window["webpackJsonp"] = window["webpackJsonp"] || []).push([[80],{

/***/ "./resources/js/form/ui/ButtonCancel.js":
/*!**********************************************!*\
  !*** ./resources/js/form/ui/ButtonCancel.js ***!
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
!(function webpackMissingModule() { var e = new Error("Cannot find module '../Data'"); e.code = 'MODULE_NOT_FOUND'; throw e; }());





var strings = new react_localization__WEBPACK_IMPORTED_MODULE_1___default.a({
  en: {
    cancel: "Cancel"
  },
  fr: {
    cancel: "Annuler"
  }
});

var ButtonCancel = function ButtonCancel() {
  return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(!(function webpackMissingModule() { var e = new Error("Cannot find module '../Data'"); e.code = 'MODULE_NOT_FOUND'; throw e; }()), null, function (_ref) {
    var item = _ref.item,
        itemChanged = _ref.itemChanged,
        unsetItem = _ref.unsetItem;
    return item && /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Box"], {
      pad: "small"
    }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Button"], {
      plain: true,
      icon: /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet_icons__WEBPACK_IMPORTED_MODULE_3__["Close"], null),
      disabled: !itemChanged && !item.id,
      label: strings.cancel,
      onClick: unsetItem
    }));
  });
};

/* harmony default export */ __webpack_exports__["default"] = (ButtonCancel);

/***/ })

}]);