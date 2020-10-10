(window["webpackJsonp"] = window["webpackJsonp"] || []).push([[77],{

/***/ "./resources/js/form/FormXX.js":
/*!*************************************!*\
  !*** ./resources/js/form/FormXX.js ***!
  \*************************************/
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
!(function webpackMissingModule() { var e = new Error("Cannot find module './FormContext'"); e.code = 'MODULE_NOT_FOUND'; throw e; }());





var strings = new react_localization__WEBPACK_IMPORTED_MODULE_1___default.a({
  en: {
    reset: "Reset",
    save: "Save"
  },
  fr: {
    reset: "Réinitialiser",
    save: "Sauvegarder"
  }
});

var FormXX = function FormXX(_ref) {
  var children = _ref.children,
      onSubmit = _ref.onSubmit;
  return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(!(function webpackMissingModule() { var e = new Error("Cannot find module './FormContext'"); e.code = 'MODULE_NOT_FOUND'; throw e; }()), null, function (_ref2) {
    var item = _ref2.item,
        save = _ref2.save,
        load = _ref2.load,
        changed = _ref2.changed,
        setChanged = _ref2.setChanged;
    return /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Form"], {
      value: item || {},
      onReset: load,
      onChange: function onChange() {
        return setChanged(true);
      },
      onSubmit: function onSubmit(_ref3) {
        var value = _ref3.value;
        return save(value);
      }
    }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Box"], null, children), /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Box"], {
      direction: "row",
      justify: "start",
      pad: {
        vertical: 'xsmall'
      }
    }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Box"], {
      pad: "small"
    }, /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet__WEBPACK_IMPORTED_MODULE_2__["Button"], {
      plain: true,
      icon: /*#__PURE__*/react__WEBPACK_IMPORTED_MODULE_0___default.a.createElement(grommet_icons__WEBPACK_IMPORTED_MODULE_3__["Save"], null),
      disabled: !changed,
      type: "submit",
      label: strings.save
    }))));
  });
};

/* harmony default export */ __webpack_exports__["default"] = (FormXX);

/***/ })

}]);